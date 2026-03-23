package stats

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMatchDurationStatsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 对局时长统计
func NewGetMatchDurationStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMatchDurationStatsLogic {
	return &GetMatchDurationStatsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMatchDurationStatsLogic) GetMatchDurationStats(req *types.GetMatchDurationStatsReq) (resp *types.GetMatchDurationStatsResp, err error) {
	// todo: add your logic here and delete this line

	return
}
