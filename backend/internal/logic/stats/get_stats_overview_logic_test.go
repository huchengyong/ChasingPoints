package stats

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestStatsOverviewCombinesBoundedSnapshotBlocks(t *testing.T) {
	svcCtx := newStatsLogicTestSvc(t)
	if err := svcCtx.DB.AutoMigrate(&model.RankChangeLog{}); err != nil {
		t.Fatalf("prepare rank trend schema: %v", err)
	}
	completedAt := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	if err := svcCtx.DB.Create(&[]model.UserCompetitiveStats{
		{UserId: 101, GameType: 0, TotalMatches: 5, Wins: 3, Losses: 2, Revision: 9},
		{UserId: 101, GameType: 3, TotalMatches: 3, Wins: 2, Losses: 1, HighestScore: 11, DurationCount: 3, DurationSumSeconds: 360, DurationMinSeconds: 90, DurationMaxSeconds: 150},
	}).Error; err != nil {
		t.Fatalf("seed snapshots: %v", err)
	}
	if err := svcCtx.DB.Create(&model.MatchParticipantResult{MatchId: 1, UserId: 101, OpponentName: "对手", GameType: 3, MatchMode: model.MatchModeRanked, Result: 1, CompletedAt: completedAt, MatchHighScore: 11}).Error; err != nil {
		t.Fatalf("seed projection: %v", err)
	}
	if err := svcCtx.DB.Create(&model.UserOpponentStrengthBucket{UserId: 101, GameType: 3, RankBucket: "score_0_1000", Matches: 2, Wins: 1}).Error; err != nil {
		t.Fatalf("seed strength bucket: %v", err)
	}
	if err := svcCtx.DB.Create(&model.RankChangeLog{UserId: 101, MatchId: 1, ChangeType: "match_result", Result: "win", Remark: "seed", GameType: 3, AfterScore: 1010, EffectiveAt: completedAt}).Error; err != nil {
		t.Fatalf("seed rank change: %v", err)
	}

	resp, err := NewGetStatsOverviewLogic(statsLogicCtx(101), svcCtx).GetStatsOverview(&types.GetStatsOverviewReq{GameType: 3, TrendLimit: 60})
	if err != nil || !resp.Success {
		t.Fatalf("get stats overview: resp=%#v err=%v", resp, err)
	}
	for _, scope := range []string{"by_game_type", "recent_trend", "rank_score_trend", "single_high_scores", "duration", "opponent_strength", "competitive_revision"} {
		if !resp.Availability[scope] {
			t.Fatalf("expected available stats scope %q: availability=%+v errors=%+v", scope, resp.Availability, resp.PartialErrors)
		}
	}
	if resp.CompetitiveRevision != 9 || len(resp.ByGameType) != 1 || len(resp.RecentTrend) != 1 || len(resp.RankScoreTrend) != 1 || len(resp.SingleHighScores) != 1 || resp.Duration == nil || resp.Duration.AverageSeconds != 120 || len(resp.OpponentStrength) != 1 {
		t.Fatalf("unexpected overview data: %+v", resp)
	}
}

func TestStatsOverviewKeepsOtherBlocksWhenRankTrendIsUnavailable(t *testing.T) {
	svcCtx := newStatsLogicTestSvc(t)
	svcCtx.RankingModel = nil
	if err := svcCtx.DB.Create(&model.UserCompetitiveStats{UserId: 101, GameType: 0, Revision: 2}).Error; err != nil {
		t.Fatalf("seed overview snapshot: %v", err)
	}
	resp, err := NewGetStatsOverviewLogic(statsLogicCtx(101), svcCtx).GetStatsOverview(&types.GetStatsOverviewReq{})
	if err != nil || !resp.Success || resp.Availability["rank_score_trend"] || !resp.Availability["by_game_type"] || !resp.Availability["competitive_revision"] {
		t.Fatalf("optional rank block must not fail overview: resp=%#v err=%v", resp, err)
	}
}

func TestStatsOverviewRejectsMissingAuthentication(t *testing.T) {
	resp, err := NewGetStatsOverviewLogic(context.Background(), newStatsLogicTestSvc(t)).GetStatsOverview(&types.GetStatsOverviewReq{})
	if err != nil || resp.Success || len(resp.Availability) != 0 {
		t.Fatalf("unauthenticated stats overview must not return aggregate data: resp=%#v err=%v", resp, err)
	}
}

func TestStatsOverviewUsesFixedQueryCountForOneAndHundredRows(t *testing.T) {
	oneQueries := statsOverviewQueryCount(t, 1)
	hundredQueries := statsOverviewQueryCount(t, 100)
	if oneQueries != 5 || hundredQueries != 5 {
		t.Fatalf("stats overview must use five bounded business queries after schema warmup: one=%d hundred=%d", oneQueries, hundredQueries)
	}
}

func TestStatsOverviewLimitsIndependentLoadConcurrency(t *testing.T) {
	var active, maximum atomic.Int32
	blocks := make([]statsOverviewBlock, 8)
	for index := range blocks {
		blocks[index] = statsOverviewBlock{scope: fmt.Sprintf("block_%d", index), load: func() (func(*types.GetStatsOverviewResp), bool) {
			current := active.Add(1)
			for {
				previous := maximum.Load()
				if current <= previous || maximum.CompareAndSwap(previous, current) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			active.Add(-1)
			return func(*types.GetStatsOverviewResp) {}, true
		}}
	}
	loaded := loadStatsOverviewBlocks(blocks)
	if maximum.Load() > maxStatsOverviewConcurrentLoads || len(loaded) != len(blocks) {
		t.Fatalf("overview loads exceeded concurrency bound: max=%d loaded=%d", maximum.Load(), len(loaded))
	}
}

func statsOverviewQueryCount(t *testing.T, rows int) int64 {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+fmt.Sprintf("-%d?mode=memory&cache=shared", rows)), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.UserCompetitiveStats{}, &model.MatchParticipantResult{}, &model.UserOpponentStrengthBucket{}, &model.RankChangeLog{}); err != nil {
		t.Fatalf("prepare stats overview schema: %v", err)
	}
	if err := db.Create(&[]model.UserCompetitiveStats{{UserId: 101, GameType: 0, Revision: 5}, {UserId: 101, GameType: 3, DurationCount: 1, DurationSumSeconds: 60, DurationMinSeconds: 60, DurationMaxSeconds: 60}}).Error; err != nil {
		t.Fatalf("seed stats snapshots: %v", err)
	}
	now := time.Now()
	participantRows := make([]model.MatchParticipantResult, 0, rows)
	rankRows := make([]model.RankChangeLog, 0, rows)
	for index := 1; index <= rows; index++ {
		participantRows = append(participantRows, model.MatchParticipantResult{MatchId: int64(index), UserId: 101, OpponentName: "对手", GameType: 3, MatchMode: model.MatchModeRanked, Result: 1, CompletedAt: now.Add(-time.Duration(index) * time.Minute), MatchHighScore: index})
		rankRows = append(rankRows, model.RankChangeLog{UserId: 101, MatchId: int64(index), GameType: 3, ChangeType: "match_result", Result: "win", Remark: "seed", EffectiveAt: now.Add(-time.Duration(index) * time.Minute), AfterScore: 1000 + index})
	}
	if err := db.Create(&participantRows).Error; err != nil {
		t.Fatalf("seed participant rows: %v", err)
	}
	if err := db.Create(&rankRows).Error; err != nil {
		t.Fatalf("seed rank rows: %v", err)
	}
	if err := db.Create(&model.UserOpponentStrengthBucket{UserId: 101, GameType: 3, RankBucket: "score_0_1000", Matches: rows, Wins: rows / 2}).Error; err != nil {
		t.Fatalf("seed strength rows: %v", err)
	}
	rankingModel := model.NewRankingModel(db)
	if _, err := rankingModel.ListRankChangesByUserAndGameType(101, 3, 1); err != nil {
		t.Fatalf("warm rank change schema capability: %v", err)
	}
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := context.WithValue(observability.WithRequestMetrics(context.Background(), metrics), "user_id", int64(101))
	requestDB := db.WithContext(ctx)
	svcCtx := &svc.ServiceContext{DB: requestDB, CompetitiveReadModel: model.NewCompetitiveReadModel(requestDB), RankingModel: rankingModel.WithDB(requestDB), Config: config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}}}
	resp, err := NewGetStatsOverviewLogic(ctx, svcCtx).GetStatsOverview(&types.GetStatsOverviewReq{GameType: 3, TrendLimit: 100})
	if err != nil || !resp.Success {
		t.Fatalf("get stats overview: resp=%#v err=%v", resp, err)
	}
	return metrics.Snapshot().SQLCount
}
