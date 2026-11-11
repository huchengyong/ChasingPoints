package match

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

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
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetH2HStatsLogic) GetH2HStats(req *types.H2HStatsReq) (resp *types.H2HStatsResp, err error) {
	if req == nil {
		req = &types.H2HStatsReq{}
	}
	viewerUserId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.H2HStatsResp{Success: false, Message: "获取交锋统计失败"}, nil
	}
	if l.svcCtx == nil {
		return &types.H2HStatsResp{Success: false, Message: "竞技读模型不可用"}, nil
	}
	if !l.svcCtx.CompetitiveReadModelsEnabled() {
		if l.svcCtx.MatchModel == nil {
			return &types.H2HStatsResp{Success: false, Message: "竞技读模型不可用"}, nil
		}
		return l.getLegacyH2HStats(viewerUserId, req)
	}
	target, err := resolveH2HReadTarget(l.svcCtx, viewerUserId, req.TargetUserId, req.OpponentId, req.OpponentName)
	if err != nil {
		return &types.H2HStatsResp{Success: false, Message: err.Error()}, nil
	}
	if target.hidden {
		return &types.H2HStatsResp{Success: true, Message: "对方已隐藏战绩", Hidden: true}, nil
	}
	if l.svcCtx.CompetitiveReadModel == nil {
		return &types.H2HStatsResp{Success: false, Message: "竞技读模型不可用"}, nil
	}
	stats, err := l.svcCtx.CompetitiveReadModel.FindOpponentStats(target.subjectUserID, target.opponentUserID, target.opponentNameKey, 0)
	if err != nil {
		l.Logger.Errorf("获取交锋统计快照失败: %v", err)
		return &types.H2HStatsResp{Success: false, Message: "获取交锋统计失败"}, nil
	}
	result := h2hStatsPayload(stats)
	return &types.H2HStatsResp{
		Success:  true,
		Opponent: &types.H2HOpponent{Id: target.opponentUserID, Name: target.opponentName, Avatar: target.opponentAvatar},
		Stats:    result,
	}, nil
}

func (l *GetH2HStatsLogic) getLegacyH2HStats(viewerUserID int64, req *types.H2HStatsReq) (*types.H2HStatsResp, error) {
	target, err := resolveH2HReadTarget(l.svcCtx, viewerUserID, req.TargetUserId, req.OpponentId, req.OpponentName)
	if err != nil {
		return &types.H2HStatsResp{Success: false, Message: err.Error()}, nil
	}
	if target.hidden {
		return &types.H2HStatsResp{Success: true, Message: "对方已隐藏战绩", Hidden: true}, nil
	}
	total, wins, losses, averageDiff, maxStreak, err := l.svcCtx.MatchModel.GetH2HStatsByOpponent(target.subjectUserID, target.opponentUserID, target.opponentName)
	if err != nil {
		l.Logger.Errorf("获取历史交锋统计失败: %v", err)
		return &types.H2HStatsResp{Success: false, Message: "获取交锋统计失败"}, nil
	}
	winRate := 0.0
	if total > 0 {
		winRate = float64(wins) / float64(total) * 100
	}
	return &types.H2HStatsResp{
		Success:  true,
		Opponent: &types.H2HOpponent{Id: target.opponentUserID, Name: target.opponentName, Avatar: target.opponentAvatar},
		Stats: &types.H2HStats{
			TotalMatches: total,
			MyWins:       wins,
			OpponentWins: losses,
			WinRate:      winRate,
			AvgScoreDiff: averageDiff,
			MaxWinStreak: maxStreak,
		},
	}, nil
}
