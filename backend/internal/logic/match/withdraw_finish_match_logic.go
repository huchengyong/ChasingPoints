package match

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type WithdrawFinishMatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 撤回排位结束请求
func NewWithdrawFinishMatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WithdrawFinishMatchLogic {
	return &WithdrawFinishMatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *WithdrawFinishMatchLogic) WithdrawFinishMatch(req *types.FinishMatchActionReq) (resp *types.FinishMatchActionResp, err error) {
	return l.withdraw(req)
}
