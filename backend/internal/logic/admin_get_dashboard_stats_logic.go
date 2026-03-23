package logic

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
	// 获取总用户数
	totalUsers, err := l.svcCtx.UserModel.CountTotal()
	if err != nil {
		l.Logger.Errorf("获取总用户数失败: %v", err)
		totalUsers = 0
	}

	// 获取今日新增用户数
	todayNewUsers, err := l.svcCtx.UserModel.CountTodayNew()
	if err != nil {
		l.Logger.Errorf("获取今日新增用户数失败: %v", err)
		todayNewUsers = 0
	}

	// 获取总对局数
	totalMatches, err := l.svcCtx.MatchModel.CountTotal()
	if err != nil {
		l.Logger.Errorf("获取总对局数失败: %v", err)
		totalMatches = 0
	}

	// 获取进行中对局数
	ongoingMatches, err := l.svcCtx.MatchModel.CountOngoing()
	if err != nil {
		l.Logger.Errorf("获取进行中对局数失败: %v", err)
		ongoingMatches = 0
	}

	return &types.AdminDashboardStatsResp{
		Code:           0,
		Success:        true,
		Message:        "success",
		TotalUsers:     totalUsers,
		TodayNewUsers:  todayNewUsers,
		TotalMatches:   totalMatches,
		OngoingMatches: ongoingMatches,
	}, nil
}
