package social

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMyPostListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取我的动态
func NewGetMyPostListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMyPostListLogic {
	return &GetMyPostListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetMyPostListLogic) GetMyPostList(req *types.GetPostListReq) (resp *types.GetPostListResp, err error) {
	if !l.svcCtx.Config.SocialEnabled() {
		return &types.GetPostListResp{Success: false, Total: 0, List: []types.SocialPostInfo{}}, nil
	}

	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetPostListResp{Success: false}, nil
	}

	posts, total, err := l.svcCtx.SocialPostModel.FindPostsByUser(userIdInt, req.Page, req.PageSize)
	if err != nil {
		l.Logger.Errorf("查询我的动态失败: userId=%d err=%v", userIdInt, err)
		return &types.GetPostListResp{Success: false}, nil
	}

	list := make([]types.SocialPostInfo, 0, len(posts))
	view, err := loadSocialPostViewData(l.svcCtx, posts, userIdInt)
	if err != nil {
		l.Logger.Errorf("加载我的动态作者或点赞信息失败: %v", err)
		return &types.GetPostListResp{Success: false, List: []types.SocialPostInfo{}}, nil
	}
	for _, post := range posts {
		list = append(list, buildSocialPostInfo(post, view, userIdInt, true))
	}

	return &types.GetPostListResp{Success: true, Total: total, List: list}, nil
}
