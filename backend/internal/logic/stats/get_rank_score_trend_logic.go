package stats

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRankScoreTrendLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 段位分变化趋势
func NewGetRankScoreTrendLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRankScoreTrendLogic {
	return &GetRankScoreTrendLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetRankScoreTrendLogic) GetRankScoreTrend(req *types.GetRankScoreTrendReq) (resp *types.GetRankScoreTrendResp, err error) {
	// todo: add your logic here and delete this line

	return
}
