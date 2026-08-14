package user

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newUserStatsSnapshotTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: observability.NewGormLogger(time.Hour),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.UserCompetitiveStats{}, &model.Match{}); err != nil {
		t.Fatalf("prepare stats snapshot schema: %v", err)
	}
	return &svc.ServiceContext{
		DB:                   db,
		MatchModel:           model.NewMatchModel(db),
		CompetitiveReadModel: model.NewCompetitiveReadModel(db),
		Config:               config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}},
	}
}

func TestGetUserStatsReadsOverallCompetitiveSnapshot(t *testing.T) {
	svcCtx := newUserStatsSnapshotTestSvc(t)
	if err := svcCtx.DB.Create(&model.UserCompetitiveStats{
		UserId: 7, GameType: 0, TotalMatches: 3, Wins: 2, Losses: 1, MaxWinStreak: 2, Revision: 5,
	}).Error; err != nil {
		t.Fatalf("seed overall snapshot: %v", err)
	}

	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(newUserContext(7), metrics)
	resp, err := NewGetUserStatsLogic(ctx, svcCtx).GetUserStats()
	if err != nil || !resp.Success {
		t.Fatalf("get user stats: resp=%#v err=%v", resp, err)
	}
	if resp.TotalMatches != 3 || resp.Wins != 2 || resp.Losses != 1 || resp.MaxWinStreak != 2 || resp.WinRate != 66.6 {
		t.Fatalf("unexpected snapshot response: %#v", resp)
	}
	if sqlCount := metrics.Snapshot().SQLCount; sqlCount != 1 {
		t.Fatalf("user stats must use one snapshot query, got %d", sqlCount)
	}
}

func TestGetUserStatsFallsBackToFactsWhenEnabledSnapshotIsMissing(t *testing.T) {
	svcCtx := newUserStatsSnapshotTestSvc(t)
	win := 1
	opponentID := int64(9)
	if err := svcCtx.DB.Create(&model.Match{Id: 1, UserId: 8, OpponentId: &opponentID, GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &win, MatchTime: time.Now()}).Error; err != nil {
		t.Fatalf("seed legacy match: %v", err)
	}
	resp, err := NewGetUserStatsLogic(context.WithValue(context.Background(), "user_id", int64(8)), svcCtx).GetUserStats()
	if err != nil || !resp.Success {
		t.Fatalf("get missing snapshot: resp=%#v err=%v", resp, err)
	}
	if resp.TotalMatches != 1 || resp.Wins != 1 || resp.Losses != 0 || resp.MaxWinStreak != 1 || resp.WinRate != 100 {
		t.Fatalf("missing snapshot must use facts instead of successful zero values: %#v", resp)
	}
}
