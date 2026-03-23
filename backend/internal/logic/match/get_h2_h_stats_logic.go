package match

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetH2HStatsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取交锋统计
func NewGetH2HStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetH2HStatsLogic {
	return &GetH2HStatsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetH2HStatsLogic) GetH2HStats(req *types.H2HStatsReq) (resp *types.H2HStatsResp, err error) {
	// todo: add your logic here and delete this line

	return
}
