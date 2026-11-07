package logic

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

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLoadCurrentCompetitiveProfileReadsOverallAndGameSnapshots(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.UserCompetitiveStats{}); err != nil {
		t.Fatalf("prepare stats schema: %v", err)
	}
	if err := db.Create(&[]model.UserCompetitiveStats{
		{UserId: 9, GameType: 0, TotalMatches: 8, Wins: 6, Revision: 12},
		{UserId: 9, GameType: 1, HighestScore: 91, HighestBreak: 76},
		{UserId: 9, GameType: 3, HighestScore: 11},
	}).Error; err != nil {
		t.Fatalf("seed snapshots: %v", err)
	}
	svcCtx := &svc.ServiceContext{
		CompetitiveReadModel: model.NewCompetitiveReadModel(db),
		Config:               config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}},
	}

	snooker, err := LoadCurrentCompetitiveProfile(svcCtx, 9, 1)
	if err != nil {
		t.Fatalf("load snooker profile: %v", err)
	}
	if snooker.WinRate != 75 || snooker.MaxScore != 76 || snooker.Revision != 12 {
		t.Fatalf("unexpected snooker snapshot profile: %+v", snooker)
	}
	nineBall, err := LoadCurrentCompetitiveProfile(svcCtx, 9, 3)
	if err != nil {
		t.Fatalf("load nine-ball profile: %v", err)
	}
	if nineBall.WinRate != 75 || nineBall.MaxScore != 11 || nineBall.Revision != 12 {
		t.Fatalf("unexpected nine-ball snapshot profile: %+v", nineBall)
	}
}

func TestLoadCurrentCompetitiveProfileUsesLegacyFactsWhileReadModelDisabled(t *testing.T) {
	db := prepareLegacyCompetitiveProfileTestDB(t)
	if err := db.Create(&[]model.UserCompetitiveStats{
		{UserId: 9, GameType: 0, TotalMatches: 10, Wins: 9, Revision: 99},
		{UserId: 9, GameType: 3, HighestScore: 99},
	}).Error; err != nil {
		t.Fatalf("seed disabled snapshots: %v", err)
	}
	svcCtx := &svc.ServiceContext{
		MatchModel:           model.NewMatchModel(db),
		CompetitiveReadModel: model.NewCompetitiveReadModel(db),
	}

	profile, err := LoadCurrentCompetitiveProfile(svcCtx, 9, 3)
	if err != nil {
		t.Fatalf("load disabled profile: %v", err)
	}
	if profile.WinRate != 50 || profile.MaxScore != 15 || profile.Revision != 0 {
		t.Fatalf("disabled read model must use legacy facts, got %+v", profile)
	}
}

func TestLoadCurrentCompetitiveProfileFallsBackWhenEnabledSnapshotIsIncomplete(t *testing.T) {
	db := prepareLegacyCompetitiveProfileTestDB(t)
	if err := db.Create(&model.UserCompetitiveStats{UserId: 9, GameType: 0, TotalMatches: 10, Wins: 9, Revision: 99}).Error; err != nil {
		t.Fatalf("seed incomplete snapshot: %v", err)
	}
	if err := seedCompetitiveProfileRankings(db, 9, 1, 1); err != nil {
		t.Fatalf("seed incomplete ranking facts: %v", err)
	}
	svcCtx := &svc.ServiceContext{
		MatchModel:           model.NewMatchModel(db),
		RankingModel:         model.NewRankingModel(db),
		CompetitiveReadModel: model.NewCompetitiveReadModel(db),
		Config:               config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}},
	}

	profile, err := LoadCurrentCompetitiveProfile(svcCtx, 9, 3)
	if err != nil {
		t.Fatalf("load incomplete profile: %v", err)
	}
	if profile.WinRate != 50 || profile.MaxScore != 15 || profile.Revision != 0 {
		t.Fatalf("incomplete snapshot must use bounded facts, got %+v", profile)
	}
}

func TestLoadCurrentCompetitiveProfileIncompleteSnapshotFallbackIsBounded(t *testing.T) {
	oneQueries, oneRows := boundedCompetitiveProfileFallbackMetrics(t, 1)
	manyQueries, manyRows := boundedCompetitiveProfileFallbackMetrics(t, 250)
	if oneQueries != 4 || manyQueries != 4 {
		t.Fatalf("fallback query count must stay fixed: one=%d many=%d", oneQueries, manyQueries)
	}
	if oneRows != 3 || manyRows != 3 {
		t.Fatalf("fallback must return fixed rows instead of loading history: one=%d many=%d", oneRows, manyRows)
	}
}

func TestLoadCurrentCompetitiveProfileIncompleteSnookerSnapshotUsesBoundedActions(t *testing.T) {
	db := prepareCompetitiveProfileTestDB(t)
	win := 1
	completedAt := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	matches := make([]model.Match, 0, currentCompetitiveProfileFallbackMatchLimit+25)
	actions := make([]model.MatchAction, 0, len(matches)*2)
	for index := 1; index <= currentCompetitiveProfileFallbackMatchLimit+25; index++ {
		matchID := int64(index)
		matches = append(matches, model.Match{
			Id: matchID, UserId: 9, OpponentName: fmt.Sprintf("对手-%d", index), GameType: 1,
			MatchMode: model.MatchModeRanked, Status: 2, Result: &win,
			MyScore: index, MatchTime: completedAt.Add(time.Duration(index) * time.Minute), CompletedAt: &completedAt,
		})
		if index <= currentCompetitiveProfileFallbackMatchLimit {
			actions = append(actions,
				model.MatchAction{MatchId: matchID, RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 1},
				model.MatchAction{MatchId: matchID, RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: index},
			)
		}
	}
	if err := db.CreateInBatches(matches, 250).Error; err != nil {
		t.Fatalf("seed snooker matches: %v", err)
	}
	if err := db.CreateInBatches(actions, 250).Error; err != nil {
		t.Fatalf("seed snooker actions: %v", err)
	}
	metrics := observability.NewRequestMetrics(time.Now())
	requestDB, queryRows := requestDBWithRowCounter(t, db, metrics)
	profile, err := LoadCurrentCompetitiveProfile(&svc.ServiceContext{
		MatchModel:           model.NewMatchModel(requestDB),
		RankingModel:         model.NewRankingModel(requestDB),
		CompetitiveReadModel: model.NewCompetitiveReadModel(requestDB),
		Config:               config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}},
	}, 9, 1)
	if err != nil {
		t.Fatalf("load bounded snooker fallback: %v", err)
	}
	if profile.MaxScore != currentCompetitiveProfileFallbackMatchLimit+1 {
		t.Fatalf("unexpected bounded snooker max score: %+v", profile)
	}
	snapshot := metrics.Snapshot()
	if snapshot.SQLCount != 5 {
		t.Fatalf("snooker fallback query count must stay fixed: %+v", snapshot)
	}
	maxRows := int64(1 + currentCompetitiveProfileFallbackMatchLimit*4)
	if rows := queryRows.Load(); rows > maxRows {
		t.Fatalf("snooker fallback must cap candidates/actions, rows=%d max=%d", rows, maxRows)
	}
}

func TestLoadCurrentCompetitiveProfileReturnsZeroWithoutFacts(t *testing.T) {
	profile, err := LoadCurrentCompetitiveProfile(&svc.ServiceContext{}, 9, 3)
	if err != nil || profile != (CurrentCompetitiveProfile{}) {
		t.Fatalf("expected empty profile without configured facts, profile=%+v err=%v", profile, err)
	}
}

func boundedCompetitiveProfileFallbackMetrics(t *testing.T, historySize int) (int64, int64) {
	t.Helper()
	db := prepareCompetitiveProfileTestDB(t, fmt.Sprintf("history-%d", historySize))
	win, loss := 1, 2
	completedAt := time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC)
	matches := make([]model.Match, 0, historySize*2)
	for index := 1; index <= historySize; index++ {
		creatorResult := loss
		if index%2 == 0 {
			creatorResult = win
		}
		matches = append(matches,
			model.Match{Id: int64(index), UserId: 9, OpponentName: "访客", GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &creatorResult, MyScore: index, MatchTime: completedAt.Add(time.Duration(index) * time.Minute), CompletedAt: &completedAt},
			model.Match{Id: int64(1000 + index), UserId: 20, OpponentId: int64Pointer(9), OpponentName: "当前用户", GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &win, OpponentScore: index + 1000, MatchTime: completedAt.Add(time.Duration(index+historySize) * time.Minute), CompletedAt: &completedAt},
		)
	}
	if err := db.Create(&matches).Error; err != nil {
		t.Fatalf("seed fallback history: %v", err)
	}
	if err := seedCompetitiveProfileRankings(db, 9, historySize, historySize); err != nil {
		t.Fatalf("seed fallback ranking facts: %v", err)
	}
	metrics := observability.NewRequestMetrics(time.Now())
	requestDB, queryRows := requestDBWithRowCounter(t, db, metrics)
	profile, err := LoadCurrentCompetitiveProfile(&svc.ServiceContext{
		MatchModel:           model.NewMatchModel(requestDB),
		RankingModel:         model.NewRankingModel(requestDB),
		CompetitiveReadModel: model.NewCompetitiveReadModel(requestDB),
		Config:               config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}},
	}, 9, 3)
	if err != nil {
		t.Fatalf("load bounded fallback: %v", err)
	}
	if profile.MaxScore != historySize+1000 {
		t.Fatalf("fallback must select the highest score from both roles: %+v", profile)
	}
	snapshot := metrics.Snapshot()
	return snapshot.SQLCount, queryRows.Load()
}

func prepareLegacyCompetitiveProfileTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := prepareCompetitiveProfileTestDB(t, "legacy")
	completedAt := time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC)
	win, loss := 1, 2
	matches := []model.Match{
		{Id: 1, UserId: 9, OpponentName: "甲", GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &win, MyScore: 15, OpponentScore: 10, MatchTime: completedAt, CompletedAt: &completedAt},
		{Id: 2, UserId: 9, OpponentName: "乙", GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &loss, MyScore: 9, OpponentScore: 11, MatchTime: completedAt.Add(time.Hour), CompletedAt: &completedAt},
	}
	if err := db.Create(&matches).Error; err != nil {
		t.Fatalf("seed legacy profile matches: %v", err)
	}
	return db
}

func prepareCompetitiveProfileTestDB(t *testing.T, suffix ...string) *gorm.DB {
	t.Helper()
	name := t.Name()
	if len(suffix) > 0 && suffix[0] != "" {
		name += "-" + suffix[0]
	}
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open profile sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Opponent{}, &model.Match{}, &model.MatchAction{}, &model.UserRanking{}, &model.UserCompetitiveStats{}); err != nil {
		t.Fatalf("prepare profile schema: %v", err)
	}
	if err := db.Create(&model.User{Id: 9, Nickname: "当前用户"}).Error; err != nil {
		t.Fatalf("seed profile user: %v", err)
	}
	return db
}

func seedCompetitiveProfileRankings(db *gorm.DB, userID int64, wins, losses int) error {
	return db.Create(&[]model.UserRanking{
		{UserId: userID, GameType: 1, TotalWins: wins, TotalLosses: losses},
		{UserId: userID, GameType: 2},
		{UserId: userID, GameType: 3},
		{UserId: userID, GameType: 4},
	}).Error
}

func requestDBWithRowCounter(t *testing.T, db *gorm.DB, metrics *observability.RequestMetrics) (*gorm.DB, *atomic.Int64) {
	t.Helper()
	rows := &atomic.Int64{}
	queryName := "test:count-query-rows:" + t.Name()
	rowName := "test:count-row-rows:" + t.Name()
	count := func(tx *gorm.DB) {
		if tx.Error == nil && tx.RowsAffected > 0 {
			rows.Add(tx.RowsAffected)
		}
	}
	if err := db.Callback().Query().After("gorm:query").Register(queryName, count); err != nil {
		t.Fatalf("register query row counter: %v", err)
	}
	if err := db.Callback().Row().After("gorm:row").Register(rowName, count); err != nil {
		t.Fatalf("register row row counter: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Query().Remove(queryName)
		_ = db.Callback().Row().Remove(rowName)
	})
	ctx := observability.WithRequestMetrics(context.Background(), metrics)
	return db.WithContext(ctx), rows
}

func int64Pointer(value int64) *int64 {
	return &value
}
