package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetDashboardStatsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取首页统计数据
func NewAdminGetDashboardStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetDashboardStatsLogic {
	return &AdminGetDashboardStatsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminGetDashboardStatsLogic) AdminGetDashboardStats() (resp *types.AdminDashboardStatsResp, err error) {
	// todo: add your logic here and delete this line

	return
}
