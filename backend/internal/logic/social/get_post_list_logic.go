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
		svcCtx: svcCtx,
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
	if err = l.svcCtx.DB.Table("follows").Where("follower_id = ?", userIdInt).Pluck("following_id", &followingIds).Error; err != nil {
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
	for _, post := range posts {
		item, ok, buildErr := buildSocialPostInfo(l.svcCtx, post, userIdInt, false)
		if buildErr != nil {
			l.Logger.Errorf("构建关注动态失败: postId=%d err=%v", post.Id, buildErr)
			continue
		}
		if !ok {
			continue
		}
		list = append(list, item)
	}

	return &types.GetPostListResp{Success: true, Total: total, List: list}, nil
}
