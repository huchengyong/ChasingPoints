package social

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPostCommentsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取动态评论
func NewGetPostCommentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPostCommentsLogic {
	return &GetPostCommentsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPostCommentsLogic) GetPostComments(req *types.GetPostCommentsReq) (resp *types.GetPostCommentsResp, err error) {
	// todo: add your logic here and delete this line

	return
}
