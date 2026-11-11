package social

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPostListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取关注的人的动态
func NewGetPostListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPostListLogic {
	return &GetPostListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetPostListLogic) GetPostList(req *types.GetPostListReq) (resp *types.GetPostListResp, err error) {
	if !l.svcCtx.Config.SocialEnabled() {
		return &types.GetPostListResp{Success: false, Total: 0, List: []types.SocialPostInfo{}}, nil
	}

	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetPostListResp{Success: false}, nil
	}

	followingIds := make([]int64, 0)
	// shortcut: 关注动态只取前 2000 个关注对象，超大关注量下按需改为分页/游标。
	if err = l.svcCtx.DB.Table("follows").Where("follower_id = ?", userIdInt).Limit(2000).Pluck("following_id", &followingIds).Error; err != nil {
		l.Logger.Errorf("查询关注列表失败: userId=%d err=%v", userIdInt, err)
		return &types.GetPostListResp{Success: false}, nil
	}
	if len(followingIds) == 0 {
		return &types.GetPostListResp{Success: true, Total: 0, List: []types.SocialPostInfo{}}, nil
	}

	posts, total, err := l.svcCtx.SocialPostModel.FindPostsByUserIds(followingIds, req.Page, req.PageSize)
	if err != nil {
		l.Logger.Errorf("查询关注动态失败: userId=%d err=%v", userIdInt, err)
		return &types.GetPostListResp{Success: false}, nil
	}

	list := make([]types.SocialPostInfo, 0, len(posts))
	view, err := loadSocialPostViewData(l.svcCtx, posts, userIdInt)
	if err != nil {
		l.Logger.Errorf("加载关注动态作者或点赞信息失败: %v", err)
		return &types.GetPostListResp{Success: false, List: []types.SocialPostInfo{}}, nil
	}
	for _, post := range posts {
		list = append(list, buildSocialPostInfo(post, view, userIdInt, false))
	}

	return &types.GetPostListResp{Success: true, Total: total, List: list}, nil
}
