package logic

import (
	"context"
	"testing"
	"time"

	achievementx "chasing_points/internal/logic/achievement"
	"chasing_points/internal/model"
)

func TestSeasonSettlementRanksOnlyQualifiedRecordsInV2Seasons(t *testing.T) {
	svcCtx := newSeasonSettlementTestSvc(t)
	svcCtx.Config.Ranking.RatingV2StartDate = "2026-07-01"
	season := model.Season{Id: 7, Name: "S7", StartDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC), Status: 1}
	if err := svcCtx.SeasonModel.Create(&season); err != nil {
		t.Fatalf("create season: %v", err)
	}
	for _, userID := range []int64{10, 20, 30, 40} {
		if err := svcCtx.UserModel.Create(&model.User{Id: userID, Nickname: "用户"}); err != nil {
			t.Fatalf("create user %d: %v", userID, err)
		}
	}
	// 20 号分数最高但未达标；10、30 号达标；40 号在另一球种达标。
	if err := svcCtx.SeasonRecordModel.CreateBatch([]model.SeasonRecord{
		{Id: 1, SeasonId: season.Id, UserId: 10, GameType: 3, StartRankScore: 1250, EndRankScore: 1600, PeakRankScore: 1600, MatchesPlayed: 9, Wins: 6, DistinctOpponents: 4, Qualified: 1},
		{Id: 2, SeasonId: season.Id, UserId: 20, GameType: 3, StartRankScore: 1250, EndRankScore: 1900, PeakRankScore: 1900, MatchesPlayed: 4, Wins: 4, DistinctOpponents: 2, Qualified: 0},
		{Id: 3, SeasonId: season.Id, UserId: 30, GameType: 3, StartRankScore: 1250, EndRankScore: 1500, PeakRankScore: 1550, MatchesPlayed: 6, Wins: 3, DistinctOpponents: 3, Qualified: 1},
		{Id: 4, SeasonId: season.Id, UserId: 40, GameType: 1, StartRankScore: 1250, EndRankScore: 1300, PeakRankScore: 1300, MatchesPlayed: 5, Wins: 3, DistinctOpponents: 3, Qualified: 1},
	}); err != nil {
		t.Fatalf("seed season records: %v", err)
	}
	service := NewSeasonSettlementService(svcCtx)
	service.sendRealtime = func(seasonRolloverRealtimeNotice) error { return nil }
	if _, err := service.SettleSeasonAt(context.Background(), season.Id, time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("settle season: %v", err)
	}

	finalRanks := map[int64]int{}
	var records []model.SeasonRecord
	if err := svcCtx.DB.Where("season_id = ?", season.Id).Find(&records).Error; err != nil {
		t.Fatalf("load records: %v", err)
	}
	for _, record := range records {
		finalRanks[record.UserId] = record.FinalRank
	}
	if finalRanks[10] != 1 || finalRanks[30] != 2 || finalRanks[20] != 0 || finalRanks[40] != 1 {
		t.Fatalf("only qualified records may be ranked per game type: %+v", finalRanks)
	}

	var titles []model.UserTitle
	if err := svcCtx.DB.Where("source_type = ? AND source_ref_id = ?", achievementx.SourceTypeSeason, season.Id).Order("user_id ASC").Find(&titles).Error; err != nil {
		t.Fatalf("load titles: %v", err)
	}
	if len(titles) != 3 || titles[0].UserId != 10 || titles[0].TitleName != "S7中式八球赛季冠军" ||
		titles[1].UserId != 30 || titles[1].TitleName != "S7中式八球赛季亚军" || titles[2].UserId != 40 || titles[2].TitleName != "S7斯诺克赛季冠军" {
		t.Fatalf("titles must only go to qualified records: %+v", titles)
	}
}

func TestSeasonSettlementKeepsRankingEveryoneInLegacySeasons(t *testing.T) {
	svcCtx := newSeasonSettlementTestSvc(t)
	svcCtx.Config.Ranking.RatingV2StartDate = "2026-08-01"
	season, _, now := seedSeasonSettlementScenario(t, svcCtx, false)
	service := NewSeasonSettlementService(svcCtx)
	service.sendRealtime = func(seasonRolloverRealtimeNotice) error { return nil }
	if _, err := service.SettleSeasonAt(context.Background(), season.Id, now); err != nil {
		t.Fatalf("settle season: %v", err)
	}
	assertSeasonSettlementRecords(t, svcCtx, season.Id)
	assertSeasonSettlementTitles(t, svcCtx, season.Id, 3)
}
