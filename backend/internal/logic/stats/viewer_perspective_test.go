package stats

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newStatsLogicTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.Match{}, &model.UserCompetitiveStats{}, &model.MatchParticipantResult{}, &model.UserOpponentStrengthBucket{}); err != nil {
		t.Fatalf("prepare stats match schema: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE user_ranking (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			game_type INTEGER NOT NULL DEFAULT 3,
			rank_score INTEGER NOT NULL DEFAULT 0,
			rank_level INTEGER NOT NULL DEFAULT 1,
			total_wins INTEGER NOT NULL DEFAULT 0,
			total_losses INTEGER NOT NULL DEFAULT 0,
			current_streak INTEGER NOT NULL DEFAULT 0,
			max_streak INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			UNIQUE(user_id, game_type)
		)
	`).Error; err != nil {
		t.Fatalf("prepare stats ranking schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:                   db,
		MatchModel:           model.NewMatchModel(db),
		CompetitiveReadModel: model.NewCompetitiveReadModel(db),
		RankingModel:         model.NewRankingModel(db),
		Config:               config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}},
	}
}

func statsLogicCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func seedStatsMatch(t *testing.T, svcCtx *svc.ServiceContext, match *model.Match) {
	t.Helper()

	if err := svcCtx.MatchModel.Create(match); err != nil {
		t.Fatalf("create match %d: %v", match.Id, err)
	}
}

func seedStatsRanking(t *testing.T, svcCtx *svc.ServiceContext, ranking *model.UserRanking) {
	t.Helper()

	if err := svcCtx.RankingModel.Create(ranking); err != nil {
		t.Fatalf("create ranking user %d game %d: %v", ranking.UserId, ranking.GameType, err)
	}
}

func seedStatsViewerPerspectiveFixtures(t *testing.T, svcCtx *svc.ServiceContext) {
	t.Helper()

	viewerID := int64(101)
	strongOpponentID := int64(202)
	weakOpponentID := int64(303)
	ignoredOpponentID := int64(404)
	win := 1
	loss := 2
	baseTime := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)

	seedStatsMatch(t, svcCtx, &model.Match{
		Id:            1,
		UserId:        viewerID,
		OpponentId:    &strongOpponentID,
		OpponentName:  "强手",
		GameType:      3,
		MyScore:       7,
		OpponentScore: 5,
		Status:        2,
		Result:        &win,
		MatchTime:     baseTime,
	})
	seedStatsMatch(t, svcCtx, &model.Match{
		Id:            2,
		UserId:        strongOpponentID,
		OpponentId:    &viewerID,
		OpponentName:  "查看者",
		GameType:      2,
		MyScore:       4,
		OpponentScore: 9,
		Status:        2,
		Result:        &loss,
		MatchTime:     baseTime.Add(1 * time.Hour),
	})
	seedStatsMatch(t, svcCtx, &model.Match{
		Id:            3,
		UserId:        weakOpponentID,
		OpponentId:    &viewerID,
		OpponentName:  "查看者",
		GameType:      2,
		MyScore:       6,
		OpponentScore: 2,
		Status:        2,
		Result:        &win,
		MatchTime:     baseTime.Add(2 * time.Hour),
	})
	seedStatsMatch(t, svcCtx, &model.Match{
		Id:            4,
		UserId:        viewerID,
		OpponentId:    &ignoredOpponentID,
		OpponentName:  "其他球种对手",
		GameType:      4,
		MyScore:       9,
		OpponentScore: 1,
		Status:        2,
		Result:        &win,
		MatchTime:     baseTime.Add(3 * time.Hour),
	})

	seedStatsRanking(t, svcCtx, &model.UserRanking{UserId: strongOpponentID, GameType: 2, RankScore: 1800, RankLevel: 3})
	seedStatsRanking(t, svcCtx, &model.UserRanking{UserId: weakOpponentID, GameType: 2, RankScore: 800, RankLevel: 1})
	seedStatsRanking(t, svcCtx, &model.UserRanking{UserId: ignoredOpponentID, GameType: 4, RankScore: 2600, RankLevel: 6})
	if err := svcCtx.DB.Create(&[]model.UserCompetitiveStats{
		{UserId: viewerID, GameType: 2, TotalMatches: 2, Wins: 1, Losses: 1, HighestScore: 9},
		{UserId: viewerID, GameType: 3, TotalMatches: 1, Wins: 1, HighestScore: 7},
		{UserId: viewerID, GameType: 4, TotalMatches: 1, Wins: 1, HighestScore: 9},
	}).Error; err != nil {
		t.Fatalf("seed competitive snapshots: %v", err)
	}
	if err := svcCtx.DB.Create(&[]model.UserOpponentStrengthBucket{
		{UserId: viewerID, GameType: 2, RankBucket: "score_1001_2000", Matches: 1, Wins: 1},
		{UserId: viewerID, GameType: 2, RankBucket: "score_0_1000", Matches: 1, Wins: 0},
	}).Error; err != nil {
		t.Fatalf("seed opponent strength buckets: %v", err)
	}
	if err := svcCtx.DB.Create(&[]model.MatchParticipantResult{
		{MatchId: 2, UserId: viewerID, GameType: 2, MatchMode: model.MatchModeRanked, Result: 1, CompletedAt: baseTime.Add(time.Hour)},
		{MatchId: 3, UserId: viewerID, GameType: 2, MatchMode: model.MatchModeRanked, Result: 2, CompletedAt: baseTime.Add(2 * time.Hour)},
	}).Error; err != nil {
		t.Fatalf("seed recent participant projections: %v", err)
	}
}

func TestStatsByGameTypeUsesViewerPerspectiveForOpponentMatches(t *testing.T) {
	svcCtx := newStatsLogicTestSvc(t)
	seedStatsViewerPerspectiveFixtures(t, svcCtx)

	resp, err := NewGetStatsByGameTypeLogic(statsLogicCtx(101), svcCtx).GetStatsByGameType()
	if err != nil {
		t.Fatalf("get stats by game type: %v", err)
	}

	var nineBallChase *types.GameTypeStats
	for i := range resp.List {
		if resp.List[i].GameType == 2 {
			nineBallChase = &resp.List[i]
			break
		}
	}
	if nineBallChase == nil {
		t.Fatalf("missing game type 2 stats in %#v", resp.List)
	}
	if nineBallChase.TotalMatches != 2 || nineBallChase.Wins != 1 || nineBallChase.Losses != 1 {
		t.Fatalf("unexpected game type 2 stats: %#v", nineBallChase)
	}
	if nineBallChase.HighestScore != 9 {
		t.Fatalf("highest score = %d, want viewer opponent_score 9", nineBallChase.HighestScore)
	}
	if nineBallChase.WinRate != 50 {
		t.Fatalf("win rate = %v, want 50", nineBallChase.WinRate)
	}
}

func TestRecentTrendUsesViewerPerspectiveForOpponentMatches(t *testing.T) {
	svcCtx := newStatsLogicTestSvc(t)
	seedStatsViewerPerspectiveFixtures(t, svcCtx)

	resp, err := NewGetRecentTrendLogic(statsLogicCtx(101), svcCtx).GetRecentTrend(&types.GetRecentTrendReq{
		GameType: 2,
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("get recent trend: %v", err)
	}
	if len(resp.List) != 2 {
		t.Fatalf("trend length = %d, want 2: %#v", len(resp.List), resp.List)
	}
	if resp.List[0].MatchId != 3 || resp.List[0].Result != 2 {
		t.Fatalf("newest opponent-created match result = %#v, want match 3 loss", resp.List[0])
	}
	if resp.List[1].MatchId != 2 || resp.List[1].Result != 1 {
		t.Fatalf("older opponent-created match result = %#v, want match 2 win", resp.List[1])
	}
}

func TestMatchDurationStatsIncludesOpponentCreatedMatches(t *testing.T) {
	svcCtx := newStatsLogicTestSvc(t)
	viewerID := int64(101)
	opponentID := int64(202)
	win := 1
	start := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Minute)

	seedStatsMatch(t, svcCtx, &model.Match{
		Id:            10,
		UserId:        opponentID,
		OpponentId:    &viewerID,
		OpponentName:  "查看者",
		GameType:      3,
		MyScore:       5,
		OpponentScore: 7,
		Status:        2,
		Result:        &win,
		MatchTime:     start,
		EndTime:       &end,
	})
	if err := svcCtx.DB.Create(&model.UserCompetitiveStats{
		UserId: viewerID, GameType: 3, TotalMatches: 1,
		DurationCount: 1, DurationSumSeconds: 120, DurationMinSeconds: 120, DurationMaxSeconds: 120,
	}).Error; err != nil {
		t.Fatalf("seed duration snapshot: %v", err)
	}

	resp, err := NewGetMatchDurationStatsLogic(statsLogicCtx(viewerID), svcCtx).GetMatchDurationStats(&types.GetMatchDurationStatsReq{
		GameType: 3,
	})
	if err != nil {
		t.Fatalf("get match duration stats: %v", err)
	}
	if resp.Stats.TotalMatches != 1 {
		t.Fatalf("total matches = %d, want opponent-created match counted", resp.Stats.TotalMatches)
	}
	if resp.Stats.AverageSeconds != 120 || resp.Stats.FastestSeconds != 120 || resp.Stats.LongestSeconds != 120 {
		t.Fatalf("duration stats = %#v, want all 120 seconds", resp.Stats)
	}
}

func TestOpponentStrengthFiltersGameTypeAndUsesActualOpponent(t *testing.T) {
	svcCtx := newStatsLogicTestSvc(t)
	seedStatsViewerPerspectiveFixtures(t, svcCtx)

	resp, err := NewGetOpponentStrengthLogic(statsLogicCtx(101), svcCtx).GetOpponentStrength(&types.GetOpponentStrengthReq{
		GameType: 2,
	})
	if err != nil {
		t.Fatalf("get opponent strength: %v", err)
	}
	if len(resp.List) != 2 {
		t.Fatalf("strength tiers = %d, want 2: %#v", len(resp.List), resp.List)
	}

	byRange := map[string]types.OpponentStrengthItem{}
	for _, item := range resp.List {
		byRange[item.RankRange] = item
	}
	if got := byRange["中级(1001-2000)"]; got.Matches != 1 || got.Wins != 1 || got.WinRate != 100 {
		t.Fatalf("middle tier = %#v, want one win", got)
	}
	if got := byRange["初级(0-1000)"]; got.Matches != 1 || got.Wins != 0 || got.WinRate != 0 {
		t.Fatalf("beginner tier = %#v, want one loss", got)
	}
	if _, ok := byRange["高级(2001+)"]; ok {
		t.Fatalf("game type 4 opponent leaked into game type 2 strength stats: %#v", resp.List)
	}
}

func TestCompetitiveStatsExcludePracticeMatches(t *testing.T) {
	svcCtx := newStatsLogicTestSvc(t)
	viewerID := int64(101)
	opponentID := int64(202)
	win := 1
	for _, match := range []*model.Match{
		{
			Id: 21, UserId: viewerID, OpponentId: &opponentID, OpponentName: "对手", GameType: 3,
			MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic, MyScore: 7, OpponentScore: 5,
			Status: 2, Result: &win, MatchTime: time.Now(),
		},
		{
			Id: 22, UserId: viewerID, OpponentId: &opponentID, OpponentName: "对手", GameType: 3,
			MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate, MyScore: 9, OpponentScore: 1,
			Status: 2, Result: &win, MatchTime: time.Now().Add(time.Minute),
		},
	} {
		seedStatsMatch(t, svcCtx, match)
	}

	if err := svcCtx.DB.Create(&model.UserCompetitiveStats{
		UserId: viewerID, GameType: 3, TotalMatches: 1, Wins: 1, HighestScore: 7,
	}).Error; err != nil {
		t.Fatalf("seed competitive snapshot: %v", err)
	}
	resp, err := NewGetStatsByGameTypeLogic(statsLogicCtx(viewerID), svcCtx).GetStatsByGameType()
	if err != nil {
		t.Fatalf("get competitive stats: %v", err)
	}
	if len(resp.List) != 1 || resp.List[0].TotalMatches != 1 || resp.List[0].Wins != 1 || resp.List[0].HighestScore != 7 {
		t.Fatalf("practice match leaked into game type stats: %#v", resp.List)
	}
}
