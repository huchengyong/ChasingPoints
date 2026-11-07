package season

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/ranking"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestV2SeasonReadsRankOnlyQualifiedRecords(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Season{}, &model.SeasonRecord{}, &model.User{}, &model.UserCompetitiveStats{}); err != nil {
		t.Fatalf("prepare schema: %v", err)
	}
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	if err := db.Create(&model.Season{Id: 1, Name: "S9", StartDate: start, EndDate: now.AddDate(0, 1, 0), Status: 1}).Error; err != nil {
		t.Fatalf("seed season: %v", err)
	}
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "我"}, {Id: 2, Nickname: "达标甲"}, {Id: 3, Nickname: "达标乙"}, {Id: 4, Nickname: "高分未达标"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Create(&[]model.SeasonRecord{
		{SeasonId: 1, UserId: 1, GameType: 3, StartRankScore: 1250, EndRankScore: 1500, PeakRankScore: 1520, MatchesPlayed: 3, Wins: 3, DistinctOpponents: 2},
		{SeasonId: 1, UserId: 2, GameType: 3, StartRankScore: 1250, EndRankScore: 1300, PeakRankScore: 1350, MatchesPlayed: 6, Wins: 4, DistinctOpponents: 3, Qualified: 1},
		{SeasonId: 1, UserId: 3, GameType: 3, StartRankScore: 1250, EndRankScore: 1200, PeakRankScore: 1300, MatchesPlayed: 8, Wins: 3, DistinctOpponents: 5, Qualified: 1},
		{SeasonId: 1, UserId: 4, GameType: 3, StartRankScore: 1250, EndRankScore: 1800, PeakRankScore: 1800, MatchesPlayed: 4, Wins: 4, DistinctOpponents: 4},
	}).Error; err != nil {
		t.Fatalf("seed records: %v", err)
	}
	svcCtx := &svc.ServiceContext{
		SeasonModel:          model.NewSeasonModel(db),
		SeasonRecordModel:    model.NewSeasonRecordModel(db),
		CompetitiveReadModel: model.NewCompetitiveReadModel(db),
		Config: config.Config{
			CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"},
			Ranking:              config.RankingConfig{RatingV2StartDate: start.Format(time.DateOnly)},
		},
	}
	asUser := func(userID int64) context.Context { return context.WithValue(context.Background(), "user_id", userID) }

	board, err := NewGetSeasonLeaderboardLogic(context.Background(), svcCtx).GetSeasonLeaderboard(&types.GetSeasonLeaderboardReq{SeasonId: 1, GameType: 3, Page: 1, PageSize: 20})
	if err != nil || !board.Success || board.Season == nil || board.Season.QualifyMinMatches != ranking.SeasonQualifyMinMatches || board.Season.QualifyMinOpponents != ranking.SeasonQualifyMinOpponents {
		t.Fatalf("leaderboard must expose qualification thresholds: resp=%+v err=%v", board, err)
	}
	wantUsers := []int64{2, 3, 4, 1}
	wantRanks := []int{1, 2, 0, 0}
	for index, item := range board.List {
		if item.UserId != wantUsers[index] || item.Rank != wantRanks[index] || item.Qualified != (wantRanks[index] > 0) {
			t.Fatalf("qualified records must come first and only they get ranks: %+v", board.List)
		}
	}

	mine, err := NewGetSeasonOverviewLogic(asUser(1), svcCtx).GetSeasonOverview(&types.GetSeasonOverviewReq{GameType: 3})
	if err != nil || !mine.Success || mine.Record == nil || mine.Record.Qualified || mine.Record.DistinctOpponents != 2 || mine.MyRank != 0 || !mine.Availability["my_rank"] {
		t.Fatalf("unqualified viewer must get progress without a rank: resp=%+v err=%v", mine, err)
	}
	second, err := NewGetSeasonOverviewLogic(asUser(3), svcCtx).GetSeasonOverview(&types.GetSeasonOverviewReq{GameType: 3})
	if err != nil || !second.Success || second.Record == nil || !second.Record.Qualified || second.MyRank != 2 {
		t.Fatalf("qualified viewer rank must match the leaderboard: resp=%+v err=%v", second, err)
	}

	record, err := NewGetMySeasonRecordLogic(asUser(4), svcCtx).GetMySeasonRecord(&types.GetMySeasonRecordReq{SeasonId: 1, GameType: 3})
	if err != nil || !record.Success || record.Record == nil || record.Record.Qualified || record.Record.DistinctOpponents != 4 {
		t.Fatalf("my season record must expose qualification: resp=%+v err=%v", record, err)
	}
}

func TestLegacySeasonReadsTreatEveryRecordAsQualified(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Season{}, &model.SeasonRecord{}, &model.User{}); err != nil {
		t.Fatalf("prepare schema: %v", err)
	}
	now := time.Now().UTC()
	if err := db.Create(&model.Season{Id: 1, Name: "S1", StartDate: now.AddDate(0, -1, 0), EndDate: now.AddDate(0, 1, 0), Status: 1}).Error; err != nil {
		t.Fatalf("seed season: %v", err)
	}
	if err := db.Create(&[]model.SeasonRecord{
		{SeasonId: 1, UserId: 1, GameType: 3, EndRankScore: 900, MatchesPlayed: 1, Wins: 0},
		{SeasonId: 1, UserId: 2, GameType: 3, EndRankScore: 1200, MatchesPlayed: 1, Wins: 1},
	}).Error; err != nil {
		t.Fatalf("seed records: %v", err)
	}
	svcCtx := &svc.ServiceContext{SeasonModel: model.NewSeasonModel(db), SeasonRecordModel: model.NewSeasonRecordModel(db)}
	board, err := NewGetSeasonLeaderboardLogic(context.Background(), svcCtx).GetSeasonLeaderboard(&types.GetSeasonLeaderboardReq{SeasonId: 1, GameType: 3, Page: 1, PageSize: 20})
	if err != nil || !board.Success || board.Season.QualifyMinMatches != 0 || len(board.List) != 2 {
		t.Fatalf("legacy leaderboard: resp=%+v err=%v", board, err)
	}
	for index, item := range board.List {
		if item.Rank != index+1 || !item.Qualified {
			t.Fatalf("legacy seasons must rank everyone: %+v", board.List)
		}
	}
}
