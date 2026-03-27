package user

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserStatsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取用户统计数据
func NewGetUserStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserStatsLogic {
	return &GetUserStatsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserStatsLogic) GetUserStats() (resp *types.UserStatsResp, err error) {
	// 获取用户ID
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return &types.UserStatsResp{Success: false}, nil
	}

	// 获取用户统计数据
	stats, err := l.svcCtx.MatchModel.GetUserStats(userId)
	if err != nil {
		l.Logger.Errorf("获取用户统计失败: %v", err)
		return &types.UserStatsResp{Success: false}, nil
	}

	// 计算胜率
	var winRate float64
	if stats.TotalMatches > 0 {
		winRate = float64(stats.Wins) / float64(stats.TotalMatches) * 100
		// 保留一位小数
		winRate = float64(int(winRate*10)) / 10
	}

	return &types.UserStatsResp{
		Success:      true,
		TotalMatches: stats.TotalMatches,
		Wins:         stats.Wins,
		Losses:       stats.Losses,
		WinRate:      winRate,
		MaxWinStreak: stats.MaxWinStreak,
	}, nil
}
