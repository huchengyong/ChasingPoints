package user

import (
	"context"

	"chasing_points/internal/model"
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
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetUserStatsLogic) GetUserStats() (resp *types.UserStatsResp, err error) {
	// 获取用户ID
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return &types.UserStatsResp{Success: false}, nil
	}

	if l.svcCtx == nil {
		return &types.UserStatsResp{Success: false}, nil
	}
	if !l.svcCtx.CompetitiveReadModelsEnabled() {
		if l.svcCtx.MatchModel == nil {
			return &types.UserStatsResp{Success: false}, nil
		}
		return l.getLegacyUserStats(userId)
	}
	if l.svcCtx.DB == nil || l.svcCtx.CompetitiveReadModel == nil {
		return &types.UserStatsResp{Success: false}, nil
	}
	stats, err := l.svcCtx.CompetitiveReadModel.FindStatsWithTx(l.svcCtx.DB.WithContext(l.ctx), userId, 0)
	if err != nil {
		l.Logger.Errorf("获取竞技统计快照失败: %v", err)
		return &types.UserStatsResp{Success: false}, nil
	}
	if stats == nil {
		// A missing snapshot after the read gate opens is a recoverable safety
		// condition, not authoritative zero statistics. Fall back to facts.
		if l.svcCtx.MatchModel == nil {
			return &types.UserStatsResp{Success: false}, nil
		}
		return l.getLegacyUserStats(userId)
	}

	return buildUserStatsFromSnapshot(stats), nil
}

func buildUserStatsFromSnapshot(stats *model.UserCompetitiveStats) *types.UserStatsResp {
	if stats == nil {
		return &types.UserStatsResp{Success: false}
	}
	winRate := 0.0
	if stats.TotalMatches > 0 {
		winRate = float64(int(float64(stats.Wins)/float64(stats.TotalMatches)*1000)) / 10
	}
	return &types.UserStatsResp{
		Success:      true,
		TotalMatches: stats.TotalMatches,
		Wins:         stats.Wins,
		Losses:       stats.Losses,
		WinRate:      winRate,
		MaxWinStreak: stats.MaxWinStreak,
	}
}

func (l *GetUserStatsLogic) getLegacyUserStats(userID int64) (*types.UserStatsResp, error) {
	stats, err := l.svcCtx.MatchModel.GetUserStats(userID)
	if err != nil {
		l.Logger.Errorf("获取历史竞技统计失败: %v", err)
		return &types.UserStatsResp{Success: false}, nil
	}
	winRate := 0.0
	if stats.TotalMatches > 0 {
		winRate = float64(int(float64(stats.Wins)/float64(stats.TotalMatches)*1000)) / 10
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
