package social

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPublicPostsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取广场动态
func NewGetPublicPostsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPublicPostsLogic {
	return &GetPublicPostsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPublicPostsLogic) GetPublicPosts(req *types.GetPostListReq) (resp *types.GetPostListResp, err error) {
	if !l.svcCtx.Config.SocialEnabled() {
		return &types.GetPostListResp{Success: false, Total: 0, List: []types.SocialPostInfo{}}, nil
	}

	userIdInt, err := utils.GetOptionalUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取可选用户ID失败: %v", err)
		return &types.GetPostListResp{Success: false, List: []types.SocialPostInfo{}}, nil
	}

	posts, total, err := l.svcCtx.SocialPostModel.FindPublicPosts(req.Page, req.PageSize)
	if err != nil {
		l.Logger.Errorf("查询广场动态失败: %v", err)
		return &types.GetPostListResp{Success: false, List: []types.SocialPostInfo{}}, nil
	}

	list := make([]types.SocialPostInfo, 0, len(posts))
	for _, post := range posts {
		item, ok, buildErr := buildSocialPostInfo(l.svcCtx, post, userIdInt, false)
		if buildErr != nil {
			l.Logger.Errorf("构建广场动态失败: postId=%d err=%v", post.Id, buildErr)
			continue
		}
		if !ok {
			continue
		}
		list = append(list, item)
	}

	return &types.GetPostListResp{Success: true, Total: total, List: list}, nil
}
