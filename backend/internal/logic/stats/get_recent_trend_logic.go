package stats

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRecentTrendLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 近N场胜率趋势
func NewGetRecentTrendLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRecentTrendLogic {
	return &GetRecentTrendLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetRecentTrendLogic) GetRecentTrend(req *types.GetRecentTrendReq) (resp *types.GetRecentTrendResp, err error) {
	// todo: add your logic here and delete this line

	return
}
