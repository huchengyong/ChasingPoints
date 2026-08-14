package match

import (
	"context"
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

func TestGetMatchListUsesParticipantProjectionPagination(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: observability.NewGormLogger(time.Hour),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Match{}, &model.MatchParticipantResult{}); err != nil {
		t.Fatalf("prepare list schema: %v", err)
	}
	base := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	matches := make([]model.Match, 0, 120)
	projections := make([]model.MatchParticipantResult, 0, 120)
	for i := 1; i <= 120; i++ {
		matchID := int64(i)
		completedAt := base
		matches = append(matches, model.Match{
			Id: matchID, UserId: 101, GameType: 3, MatchMode: model.MatchModeRanked,
			Visibility: model.MatchVisibilityPublic, FinishState: model.FinishStateNone, Status: 2,
			MatchTime: completedAt.Add(-time.Minute), CompletedAt: &completedAt,
		})
		result := 1
		if i%2 == 0 {
			result = 2
		}
		projections = append(projections, model.MatchParticipantResult{
			MatchId: matchID, UserId: 101, OpponentUserId: 202, OpponentNameKey: "user:202", OpponentName: "对手",
			GameType: 3, MatchMode: model.MatchModeRanked, Result: result, MyScore: 7, OpponentScore: 5,
			CompletedAt: completedAt,
		})
	}
	if err := db.Create(&matches).Error; err != nil {
		t.Fatalf("seed matches: %v", err)
	}
	if err := db.Create(&projections).Error; err != nil {
		t.Fatalf("seed participant projections: %v", err)
	}
	svcCtx := &svc.ServiceContext{DB: db, CompetitiveReadModel: model.NewCompetitiveReadModel(db), Config: config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}}}
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.WithValue(context.Background(), "user_id", int64(101)), metrics)

	resp, err := NewGetMatchListLogic(ctx, svcCtx).GetMatchList(&types.MatchListReq{Page: 2, PageSize: 20, GameType: 3, Result: 1})
	if err != nil || !resp.Success {
		t.Fatalf("get match list: resp=%#v err=%v", resp, err)
	}
	if resp.Total != 60 || len(resp.List) != 20 || resp.List[0].Id != 79 || resp.List[0].Result != 1 {
		t.Fatalf("unexpected projection page: %#v", resp)
	}
	if sqlCount := metrics.Snapshot().SQLCount; sqlCount != 2 {
		t.Fatalf("match list must use one COUNT and one page query, got %d", sqlCount)
	}
}
