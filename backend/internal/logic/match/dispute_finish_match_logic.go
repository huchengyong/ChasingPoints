package match

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DisputeFinishMatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 对排位结束请求提出异议
func NewDisputeFinishMatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisputeFinishMatchLogic {
	return &DisputeFinishMatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DisputeFinishMatchLogic) DisputeFinishMatch(req *types.FinishMatchActionReq) (resp *types.FinishMatchActionResp, err error) {
	return l.dispute(req)
}
