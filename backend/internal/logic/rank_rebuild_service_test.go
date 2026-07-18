package logic

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func int64Ptr(v int64) *int64 {
	return &v
}

func TestUpdateReplaySettlementStateCountsDrawForSameOpponentDailyLimit(t *testing.T) {
	dayStart := time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC)
	match := &model.Match{
		Id:         11,
		UserId:     8,
		OpponentId: int64Ptr(18),
		GameType:   3,
	}

	dailyPositiveGains := make(map[rankDailyGainKey]int)
	sameOpponentDailyCounts := make(map[rankPairDailyKey]int)

	updateReplaySettlementState(dailyPositiveGains, sameOpponentDailyCounts, match, dayStart, 0, 0)

	key := buildRankPairDailyKey(8, 18, 3, dayStart)
	if sameOpponentDailyCounts[key] != 1 {
		t.Fatalf("expected draw to count toward same-opponent daily limit, got %d", sameOpponentDailyCounts[key])
	}
}

func TestUpdateReplaySettlementStateOnlyAccumulatesPositiveDailyGain(t *testing.T) {
	dayStart := time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC)
	match := &model.Match{
		Id:         12,
		UserId:     8,
		OpponentId: int64Ptr(18),
		GameType:   3,
	}

	dailyPositiveGains := make(map[rankDailyGainKey]int)
	sameOpponentDailyCounts := make(map[rankPairDailyKey]int)

	updateReplaySettlementState(dailyPositiveGains, sameOpponentDailyCounts, match, dayStart, 28, -2)

	player1Key := rankDailyGainKey{UserId: 8, GameType: 3, DayStart: dayStart}
	player2Key := rankDailyGainKey{UserId: 18, GameType: 3, DayStart: dayStart}
	if dailyPositiveGains[player1Key] != 28 {
		t.Fatalf("expected player1 positive gain 28, got %d", dailyPositiveGains[player1Key])
	}
	if dailyPositiveGains[player2Key] != 0 {
		t.Fatalf("expected player2 non-positive gain to be ignored, got %d", dailyPositiveGains[player2Key])
	}
}

func TestRankRebuildGrantsTraceableSeasonTitlesIdempotently(t *testing.T) {
	svcCtx := newRankRebuildSeasonTitleTestSvc(t)
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC)
	if err := svcCtx.SeasonModel.Create(&model.Season{
		Id:        9001,
		Name:      "S1",
		StartDate: start,
		EndDate:   end,
		Status:    2,
	}); err != nil {
		t.Fatalf("seed season: %v", err)
	}
	opponentID := int64(1002)
	result := 1
	matchTime := time.Date(2026, 4, 12, 20, 0, 0, 0, time.UTC)
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id:            9101,
		UserId:        1001,
		OpponentId:    &opponentID,
		OpponentName:  "选手乙",
		GameType:      3,
		MyScore:       1,
		OpponentScore: 0,
		Status:        2,
		Result:        &result,
		MatchTime:     matchTime,
		EndTime:       &matchTime,
	}); err != nil {
		t.Fatalf("seed match: %v", err)
	}
	practiceResult := 1
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id:            9102,
		UserId:        1001,
		OpponentId:    &opponentID,
		OpponentName:  "选手乙",
		GameType:      3,
		MatchMode:     model.MatchModePractice,
		Visibility:    model.MatchVisibilityPrivate,
		MyScore:       9,
		OpponentScore: 0,
		Status:        2,
		Result:        &practiceResult,
		MatchTime:     matchTime.Add(time.Hour),
		EndTime:       func() *time.Time { value := matchTime.Add(time.Hour); return &value }(),
	}); err != nil {
		t.Fatalf("seed practice match: %v", err)
	}

	summary, err := NewRankRebuildService(svcCtx).Rebuild(context.Background())
	if err != nil {
		t.Fatalf("rebuild ranks: %v", err)
	}
	if summary.MatchesTotal != 1 || summary.RankLogsTotal != 2 {
		t.Fatalf("practice match leaked into ranking rebuild summary: %+v", summary)
	}
	if summary.SeasonRecords != 2 {
		t.Fatalf("expected 2 season records, got %d", summary.SeasonRecords)
	}
	assertSeasonRecordRank(t, svcCtx, 9001, 1001, 3, 1)
	assertSeasonRecordRank(t, svcCtx, 9001, 1002, 3, 2)
	assertSeasonTitle(t, svcCtx, 1001, "S1中式八球赛季冠军", 9001, "S1")
	assertSeasonTitle(t, svcCtx, 1002, "S1中式八球赛季亚军", 9001, "S1")

	replaySummary, err := NewRankRebuildService(svcCtx).Rebuild(context.Background())
	if err != nil {
		t.Fatalf("rebuild ranks again: %v", err)
	}
	if replaySummary.SeasonRecords != 2 {
		t.Fatalf("expected replay 2 season records, got %d", replaySummary.SeasonRecords)
	}
	if countSeasonTitles(t, svcCtx, 1001, "S1中式八球赛季冠军") != 1 {
		t.Fatal("expected season champion title to stay idempotent")
	}
}

func newRankRebuildSeasonTitleTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Match{},
		&model.MatchRound{},
		&model.MatchAction{},
		&model.MatchAchievement{},
		&model.UserRanking{},
		&model.RankChangeLog{},
		&model.Season{},
		&model.SeasonRecord{},
		&model.AchievementRewardConfig{},
		&model.UserTitle{},
	); err != nil {
		t.Fatalf("prepare rank rebuild season title schema: %v", err)
	}
	if err := db.Exec("DROP TABLE IF EXISTS user_ranking").Error; err != nil {
		t.Fatalf("drop sqlite user_ranking table: %v", err)
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
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			UNIQUE(user_id, game_type)
		)
	`).Error; err != nil {
		t.Fatalf("recreate sqlite user_ranking table: %v", err)
	}

	return &svc.ServiceContext{
		DB:                db,
		MatchModel:        model.NewMatchModel(db),
		RankingModel:      model.NewRankingModel(db),
		SeasonModel:       model.NewSeasonModel(db),
		SeasonRecordModel: model.NewSeasonRecordModel(db),
		UserTitleModel:    model.NewUserTitleModel(db),
	}
}

func assertSeasonRecordRank(t *testing.T, svcCtx *svc.ServiceContext, seasonID int64, userID int64, gameType int, wantRank int) {
	t.Helper()

	record, err := svcCtx.SeasonRecordModel.FindBySeasonAndUserAndGameType(seasonID, userID, gameType)
	if err != nil {
		t.Fatalf("find season record: %v", err)
	}
	if record == nil || record.FinalRank != wantRank {
		t.Fatalf("expected user=%d final_rank=%d, got %+v", userID, wantRank, record)
	}
}

func assertSeasonTitle(t *testing.T, svcCtx *svc.ServiceContext, userID int64, titleName string, refID int64, refName string) {
	t.Helper()

	var title model.UserTitle
	if err := svcCtx.DB.Where("user_id = ? AND title_name = ?", userID, titleName).First(&title).Error; err != nil {
		t.Fatalf("find season title %s: %v", titleName, err)
	}
	if title.SourceType != "season" || title.SourceRefId != refID || title.SourceRefName != refName {
		t.Fatalf("unexpected season title source: %+v", title)
	}
}

func countSeasonTitles(t *testing.T, svcCtx *svc.ServiceContext, userID int64, titleName string) int64 {
	t.Helper()

	var total int64
	if err := svcCtx.DB.Model(&model.UserTitle{}).
		Where("user_id = ? AND title_name = ?", userID, titleName).
		Count(&total).Error; err != nil {
		t.Fatalf("count season titles: %v", err)
	}
	return total
}
