package social

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

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
	// todo: add your logic here and delete this line

	return
}
