package match

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RequestFinishMatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 发起排位结束确认
func NewRequestFinishMatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RequestFinishMatchLogic {
	return &RequestFinishMatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *RequestFinishMatchLogic) RequestFinishMatch(req *types.FinishMatchActionReq) (resp *types.FinishMatchActionResp, err error) {
	return l.request(req)
}
