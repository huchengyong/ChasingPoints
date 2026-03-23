package social

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type LikePostLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 点赞动态
func NewLikePostLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LikePostLogic {
	return &LikePostLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LikePostLogic) LikePost(req *types.PostIdReq) (resp *types.CommonResp, err error) {
	// todo: add your logic here and delete this line

	return
}
