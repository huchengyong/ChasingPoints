package match

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type EndRoundLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 结束一局
func NewEndRoundLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EndRoundLogic {
	return &EndRoundLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EndRoundLogic) EndRound(req *types.EndRoundReq) (resp *types.EndRoundResp, err error) {
	// todo: add your logic here and delete this line

	return
}
