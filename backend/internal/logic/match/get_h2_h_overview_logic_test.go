package match

import (
	"context"
	"fmt"
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

func TestH2HOverviewRejectsMissingAuthentication(t *testing.T) {
	resp, err := NewGetH2HOverviewLogic(context.Background(), &svc.ServiceContext{}).GetH2HOverview(&types.H2HOverviewReq{OpponentId: 2})
	if err != nil || resp.Success || resp.Opponent != nil {
		t.Fatalf("unauthenticated H2H overview must not return data: resp=%#v err=%v", resp, err)
	}
}

func TestH2HOverviewReturnsTargetStatsAndFirstHistoryPage(t *testing.T) {
	svcCtx := newMatchLogicTestSvc(t)
	if err := svcCtx.DB.AutoMigrate(&model.UserCompetitiveStats{}); err != nil {
		t.Fatalf("prepare competitive revision schema: %v", err)
	}
	seedTargetH2HFixtures(t, svcCtx)
	if err := svcCtx.DB.Create(&model.UserCompetitiveStats{UserId: 202, GameType: 0, Revision: 6}).Error; err != nil {
		t.Fatalf("seed revision: %v", err)
	}

	resp, err := NewGetH2HOverviewLogic(matchLogicCtx(101), svcCtx).GetH2HOverview(&types.H2HOverviewReq{TargetUserId: 202, OpponentId: 404, GameType: 0, PageSize: 1})
	if err != nil || !resp.Success || resp.Opponent == nil || resp.Stats == nil {
		t.Fatalf("get h2h overview: resp=%#v err=%v", resp, err)
	}
	if resp.Opponent.Id != 404 || resp.Stats.TotalMatches != 2 || resp.Stats.MyWins != 1 || resp.Total != 2 || len(resp.List) != 1 || !resp.HasMore || resp.CompetitiveRevision != 6 {
		t.Fatalf("unexpected h2h overview: %+v", resp)
	}
	for _, scope := range []string{"opponent", "stats", "history", "competitive_revision"} {
		if !resp.Availability[scope] {
			t.Fatalf("expected available overview scope %q: %+v", scope, resp)
		}
	}
	legacy, legacyErr := NewGetH2HStatsLogic(matchLogicCtx(101), svcCtx).GetH2HStats(&types.H2HStatsReq{TargetUserId: 202, OpponentId: 404})
	if legacyErr != nil || !legacy.Success || legacy.Stats == nil || legacy.Stats.TotalMatches != resp.Stats.TotalMatches || legacy.Stats.MyWins != resp.Stats.MyWins {
		t.Fatalf("overview and legacy H2H stats must share snapshots: overview=%+v legacy=%+v err=%v", resp.Stats, legacy, legacyErr)
	}
}

func TestH2HOverviewKeepsStatsWhenDateBlockIsInvalid(t *testing.T) {
	svcCtx := newMatchLogicTestSvc(t)
	if err := svcCtx.DB.AutoMigrate(&model.UserCompetitiveStats{}); err != nil {
		t.Fatalf("prepare revision schema: %v", err)
	}
	seedTargetH2HFixtures(t, svcCtx)
	resp, err := NewGetH2HOverviewLogic(matchLogicCtx(101), svcCtx).GetH2HOverview(&types.H2HOverviewReq{
		TargetUserId: 202,
		OpponentId:   404,
		StartDate:    "not-a-date",
	})
	if err != nil || !resp.Success || !resp.Availability["stats"] || resp.Availability["history"] || resp.Stats == nil {
		t.Fatalf("invalid optional history date must retain H2H stats: resp=%#v err=%v", resp, err)
	}
	if len(resp.PartialErrors) != 1 || resp.PartialErrors[0].Scope != "history" {
		t.Fatalf("expected only history partial error: %+v", resp.PartialErrors)
	}
}

func TestH2HOverviewRejectsUnauthorizedTarget(t *testing.T) {
	svcCtx := newMatchLogicTestSvc(t)
	seedTargetH2HFixtures(t, svcCtx)
	resp, err := NewGetH2HOverviewLogic(matchLogicCtx(101), svcCtx).GetH2HOverview(&types.H2HOverviewReq{TargetUserId: 909, OpponentId: 404})
	if err != nil || resp.Success {
		t.Fatalf("unauthorized H2H target must fail: resp=%#v err=%v", resp, err)
	}
}

func TestH2HOverviewUsesFixedQueryCountForOneAndHundredRows(t *testing.T) {
	one := h2hOverviewQueryCount(t, 1)
	hundred := h2hOverviewQueryCount(t, 100)
	if one != 5 || hundred != 5 {
		t.Fatalf("H2H overview must use five fixed queries: one=%d hundred=%d", one, hundred)
	}
}

func TestH2HHistoryUsesStableProjectionPaginationWithFixedQueries(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.MatchParticipantResult{}); err != nil {
		t.Fatalf("prepare H2H history schema: %v", err)
	}
	if err := db.Create(&model.User{Id: 2, Nickname: "对手"}).Error; err != nil {
		t.Fatalf("seed opponent: %v", err)
	}
	completedAt := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	items := make([]model.MatchParticipantResult, 0, 120)
	for index := 1; index <= 120; index++ {
		items = append(items, model.MatchParticipantResult{MatchId: int64(index), UserId: 1, OpponentUserId: 2, OpponentNameKey: "user:2", OpponentName: "对手", GameType: 3, MatchMode: model.MatchModeRanked, Result: 1, CompletedAt: completedAt})
	}
	if err := db.Create(&items).Error; err != nil {
		t.Fatalf("seed H2H history: %v", err)
	}
	loadPage := func(page int) (*types.H2HHistoryResp, int64) {
		metrics := observability.NewRequestMetrics(time.Now())
		ctx := observability.WithRequestMetrics(context.WithValue(context.Background(), "user_id", int64(1)), metrics)
		requestDB := db.WithContext(ctx)
		resp, err := NewGetH2HHistoryLogic(ctx, &svc.ServiceContext{UserModel: model.NewUserModel(requestDB), CompetitiveReadModel: model.NewCompetitiveReadModel(requestDB), Config: config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}}}).GetH2HHistory(&types.H2HHistoryReq{OpponentId: 2, Page: page, PageSize: 20})
		if err != nil || !resp.Success || resp.Total != 120 || len(resp.List) != 20 {
			t.Fatalf("get H2H history page %d: resp=%#v err=%v", page, resp, err)
		}
		return resp, metrics.Snapshot().SQLCount
	}
	first, firstQueries := loadPage(1)
	second, secondQueries := loadPage(2)
	if firstQueries != 3 || secondQueries != 3 {
		t.Fatalf("H2H history must use target lookup plus count/page queries: first=%d second=%d", firstQueries, secondQueries)
	}
	if first.List[0].Id != 120 || first.List[19].Id != 101 || second.List[0].Id != 100 || second.List[19].Id != 81 {
		t.Fatalf("H2H history must use match id as stable tie-breaker: first=%+v second=%+v", first.List, second.List)
	}
}

func h2hOverviewQueryCount(t *testing.T, rows int) int64 {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+fmt.Sprintf("-%d?mode=memory&cache=shared", rows)), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserCompetitiveStats{}, &model.UserOpponentStats{}, &model.MatchParticipantResult{}); err != nil {
		t.Fatalf("prepare H2H query schema: %v", err)
	}
	if err := db.Create(&model.User{Id: 2, Nickname: "对手"}).Error; err != nil {
		t.Fatalf("seed opponent: %v", err)
	}
	if err := db.Create(&model.UserCompetitiveStats{UserId: 1, GameType: 0, Revision: 4}).Error; err != nil {
		t.Fatalf("seed revision: %v", err)
	}
	if err := db.Create(&model.UserOpponentStats{UserId: 1, OpponentUserId: 2, OpponentNameKey: "user:2", OpponentName: "对手", GameType: 0, TotalMatches: rows, Wins: rows}).Error; err != nil {
		t.Fatalf("seed opponent stats: %v", err)
	}
	items := make([]model.MatchParticipantResult, 0, rows)
	for index := 1; index <= rows; index++ {
		items = append(items, model.MatchParticipantResult{MatchId: int64(index), UserId: 1, OpponentUserId: 2, OpponentNameKey: "user:2", OpponentName: "对手", GameType: 3, MatchMode: model.MatchModeRanked, Result: 1, CompletedAt: time.Now().Add(-time.Duration(index) * time.Minute)})
	}
	if err := db.Create(&items).Error; err != nil {
		t.Fatalf("seed participant history: %v", err)
	}
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.WithValue(context.Background(), "user_id", int64(1)), metrics)
	requestDB := db.WithContext(ctx)
	resp, err := NewGetH2HOverviewLogic(ctx, &svc.ServiceContext{UserModel: model.NewUserModel(requestDB), CompetitiveReadModel: model.NewCompetitiveReadModel(requestDB), Config: config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}}}).GetH2HOverview(&types.H2HOverviewReq{OpponentId: 2, PageSize: 20})
	if err != nil || !resp.Success || !resp.Availability["history"] {
		t.Fatalf("get H2H overview: resp=%#v err=%v", resp, err)
	}
	return metrics.Snapshot().SQLCount
}
