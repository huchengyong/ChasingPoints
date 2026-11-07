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
	for _, post := range posts {
		item, ok, buildErr := buildSocialPostInfo(l.svcCtx, post, userIdInt, true)
		if buildErr != nil {
			l.Logger.Errorf("构建我的动态失败: postId=%d err=%v", post.Id, buildErr)
			continue
		}
		if !ok {
			continue
		}
		list = append(list, item)
	}

	return &types.GetPostListResp{Success: true, Total: total, List: list}, nil
}
