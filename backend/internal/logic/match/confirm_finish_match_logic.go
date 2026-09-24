package match

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ConfirmFinishMatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 确认排位结束请求
func NewConfirmFinishMatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConfirmFinishMatchLogic {
	return &ConfirmFinishMatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ConfirmFinishMatchLogic) ConfirmFinishMatch(req *types.FinishMatchActionReq) (resp *types.FinishMatchActionResp, err error) {
	return l.confirm(req)
}
