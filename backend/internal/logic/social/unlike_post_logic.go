package social

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UnlikePostLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 取消点赞
func NewUnlikePostLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnlikePostLogic {
	return &UnlikePostLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UnlikePostLogic) UnlikePost(req *types.PostIdReq) (resp *types.CommonResp, err error) {
	// todo: add your logic here and delete this line

	return
}
