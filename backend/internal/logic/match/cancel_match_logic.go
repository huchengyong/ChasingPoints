package match

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelMatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 取消对局
func NewCancelMatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelMatchLogic {
	return &CancelMatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CancelMatchLogic) CancelMatch(req *types.CancelMatchReq) (resp *types.CommonResp, err error) {
	// todo: add your logic here and delete this line

	return
}
