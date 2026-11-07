package stats

import (
	"context"
	"sync"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetStatsOverviewLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取竞技分析首屏概览
func NewGetStatsOverviewLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetStatsOverviewLogic {
	return &GetStatsOverviewLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetStatsOverviewLogic) GetStatsOverview(req *types.GetStatsOverviewReq) (resp *types.GetStatsOverviewResp, err error) {
	result := &types.GetStatsOverviewResp{Availability: map[string]bool{}, PartialErrors: []types.ReadPartialError{}}
	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil || l.svcCtx == nil || l.svcCtx.CompetitiveReadModel == nil {
		return result, nil
	}
	result.Success = true
	gameType, trendLimit := 3, 60
	if req != nil {
		if req.GameType > 0 {
			gameType = req.GameType
		}
		if req.TrendLimit > 0 {
			trendLimit = req.TrendLimit
		}
	}
	if trendLimit > 100 {
		trendLimit = 100
	}

	blocks := make([]statsOverviewBlock, 0, 6)
	if l.svcCtx.CompetitiveReadModelsEnabled() {
		rows, loadErr := l.svcCtx.CompetitiveReadModel.ListStats(userID)
		if loadErr != nil {
			appendStatsOverviewPartial(result, "by_game_type", "分球种统计读取失败")
			appendStatsOverviewPartial(result, "duration", "时长统计读取失败")
			appendStatsOverviewPartial(result, "competitive_revision", "竞技版本读取失败")
		} else {
			applyStatsSnapshotOverview(result, rows, gameType)
		}
	} else {
		blocks = append(blocks,
			statsOverviewBlock{
				scope: "by_game_type", message: "分球种统计读取失败",
				load: func() (func(*types.GetStatsOverviewResp), bool) {
					value, loadErr := NewGetStatsByGameTypeLogic(l.ctx, l.svcCtx).GetStatsByGameType()
					if loadErr != nil || value == nil || !value.Success {
						return nil, false
					}
					return func(resp *types.GetStatsOverviewResp) { resp.ByGameType = value.List }, true
				},
			},
			statsOverviewBlock{
				scope: "duration", message: "时长统计读取失败",
				load: func() (func(*types.GetStatsOverviewResp), bool) {
					value, loadErr := NewGetMatchDurationStatsLogic(l.ctx, l.svcCtx).GetMatchDurationStats(&types.GetMatchDurationStatsReq{GameType: gameType})
					if loadErr != nil || value == nil || !value.Success {
						return nil, false
					}
					return func(resp *types.GetStatsOverviewResp) { resp.Duration = value.Stats }, true
				},
			},
		)
		appendStatsOverviewPartial(result, "competitive_revision", "竞技读模型尚未切换")
	}
	blocks = append(blocks,
		statsOverviewBlock{
			scope: "recent_trend", message: "近期趋势读取失败",
			load: func() (func(*types.GetStatsOverviewResp), bool) {
				value, loadErr := NewGetRecentTrendLogic(l.ctx, l.svcCtx).GetRecentTrend(&types.GetRecentTrendReq{GameType: gameType, Limit: trendLimit})
				if loadErr != nil || value == nil || !value.Success {
					return nil, false
				}
				return func(resp *types.GetStatsOverviewResp) { resp.RecentTrend = value.List }, true
			},
		},
		statsOverviewBlock{
			scope: "single_high_scores", message: "最高分读取失败",
			load: func() (func(*types.GetStatsOverviewResp), bool) {
				value, loadErr := NewGetSingleHighScoreLogic(l.ctx, l.svcCtx).GetSingleHighScore(&types.GetSingleHighScoreReq{GameType: gameType, Limit: 10})
				if loadErr != nil || value == nil || !value.Success {
					return nil, false
				}
				return func(resp *types.GetStatsOverviewResp) { resp.SingleHighScores = value.List }, true
			},
		},
		statsOverviewBlock{
			scope: "opponent_strength", message: "对手强度读取失败",
			load: func() (func(*types.GetStatsOverviewResp), bool) {
				value, loadErr := NewGetOpponentStrengthLogic(l.ctx, l.svcCtx).GetOpponentStrength(&types.GetOpponentStrengthReq{GameType: gameType})
				if loadErr != nil || value == nil || !value.Success {
					return nil, false
				}
				return func(resp *types.GetStatsOverviewResp) { resp.OpponentStrength = value.List }, true
			},
		},
	)
	if l.svcCtx.RankingModel == nil {
		blocks = append(blocks, statsOverviewBlock{scope: "rank_score_trend", message: "段位服务不可用"})
	} else {
		blocks = append(blocks, statsOverviewBlock{
			scope: "rank_score_trend", message: "段位趋势读取失败",
			load: func() (func(*types.GetStatsOverviewResp), bool) {
				value, loadErr := NewGetRankScoreTrendLogic(l.ctx, l.svcCtx).GetRankScoreTrend(&types.GetRankScoreTrendReq{GameType: gameType, Limit: trendLimit})
				if loadErr != nil || value == nil || !value.Success {
					return nil, false
				}
				return func(resp *types.GetStatsOverviewResp) { resp.RankScoreTrend = value.List }, true
			},
		})
	}
	for _, block := range loadStatsOverviewBlocks(blocks) {
		if !block.ok {
			appendStatsOverviewPartial(result, block.scope, block.message)
			continue
		}
		block.apply(result)
		result.Availability[block.scope] = true
	}
	return result, nil
}

func applyStatsSnapshotOverview(resp *types.GetStatsOverviewResp, rows []model.UserCompetitiveStats, gameType int) {
	resp.ByGameType = make([]types.GameTypeStats, 0, len(rows))
	resp.Duration = &types.DurationStats{}
	for _, row := range rows {
		if row.GameType > 0 {
			resp.ByGameType = append(resp.ByGameType, types.GameTypeStats{
				GameType:     row.GameType,
				GameTypeName: GetGameTypeName(row.GameType),
				TotalMatches: row.TotalMatches,
				Wins:         row.Wins,
				Losses:       row.Losses,
				WinRate:      calculateWinRatePercent(row.Wins, row.TotalMatches),
				HighestScore: row.HighestScore,
			})
		}
		if row.GameType == 0 {
			resp.CompetitiveRevision = row.Revision
		}
		if row.GameType == gameType && row.DurationCount > 0 {
			resp.Duration = &types.DurationStats{
				AverageSeconds: int(row.DurationSumSeconds / int64(row.DurationCount)),
				FastestSeconds: int(row.DurationMinSeconds),
				LongestSeconds: int(row.DurationMaxSeconds),
				TotalMatches:   row.DurationCount,
			}
		}
	}
	resp.Availability["by_game_type"] = true
	resp.Availability["duration"] = true
	resp.Availability["competitive_revision"] = true
}

type statsOverviewBlock struct {
	scope   string
	message string
	load    func() (func(*types.GetStatsOverviewResp), bool)
}

type loadedStatsOverviewBlock struct {
	scope   string
	message string
	apply   func(*types.GetStatsOverviewResp)
	ok      bool
}

const maxStatsOverviewConcurrentLoads = 3

func loadStatsOverviewBlocks(blocks []statsOverviewBlock) []loadedStatsOverviewBlock {
	results := make([]loadedStatsOverviewBlock, len(blocks))
	semaphore := make(chan struct{}, maxStatsOverviewConcurrentLoads)
	var group sync.WaitGroup
	for index, block := range blocks {
		group.Add(1)
		go func(index int, block statsOverviewBlock) {
			defer group.Done()
			if block.load == nil {
				results[index] = loadedStatsOverviewBlock{scope: block.scope, message: block.message}
				return
			}
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			apply, ok := block.load()
			results[index] = loadedStatsOverviewBlock{scope: block.scope, message: block.message, apply: apply, ok: ok}
		}(index, block)
	}
	group.Wait()
	return results
}

func appendStatsOverviewPartial(resp *types.GetStatsOverviewResp, scope, message string) {
	resp.Availability[scope] = false
	resp.PartialErrors = append(resp.PartialErrors, types.ReadPartialError{Scope: scope, Message: message})
}
