package season

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSeasonReportLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取赛季报告
func NewGetSeasonReportLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSeasonReportLogic {
	return &GetSeasonReportLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSeasonReportLogic) GetSeasonReport(req *types.GetSeasonReportReq) (resp *types.GetSeasonReportResp, err error) {
	// todo: add your logic here and delete this line

	return
}
