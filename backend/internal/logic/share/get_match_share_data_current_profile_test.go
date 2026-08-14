package share

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

func TestMatchShareDataUsesLatestCompetitiveProfilesWithoutChangingScore(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}, &model.MatchRound{}, &model.MatchAction{}, &model.MatchAchievement{}, &model.UserCompetitiveStats{}); err != nil {
		t.Fatalf("prepare share schema: %v", err)
	}
	svcCtx := &svc.ServiceContext{
		DB:                   db,
		UserModel:            model.NewUserModel(db),
		MatchModel:           model.NewMatchModel(db),
		CompetitiveReadModel: model.NewCompetitiveReadModel(db),
		Config:               config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}},
	}
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "甲"}, {Id: 2, Nickname: "乙"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	opponentID := int64(2)
	result := 1
	completedAt := time.Date(2026, 8, 12, 19, 0, 0, 0, time.UTC)
	if err := db.Create(&model.Match{Id: 21, UserId: 1, OpponentId: &opponentID, OpponentName: "乙", GameType: 1, Status: 2, Result: &result, MyScore: 60, OpponentScore: 40, MatchTime: completedAt, CompletedAt: &completedAt}).Error; err != nil {
		t.Fatalf("seed match: %v", err)
	}
	if err := db.Create(&[]model.UserCompetitiveStats{
		{UserId: 1, GameType: 0, TotalMatches: 4, Wins: 2},
		{UserId: 1, GameType: 1, HighestBreak: 71},
		{UserId: 2, GameType: 0, TotalMatches: 4, Wins: 1},
		{UserId: 2, GameType: 1, HighestBreak: 63},
	}).Error; err != nil {
		t.Fatalf("seed profiles: %v", err)
	}

	first := loadShareForCurrentProfileTest(t, svcCtx)
	if first.MyScore != 60 || first.OpponentScore != 40 || summaryValue(first.SummaryStats, "我的最高单杆") != "71 分" {
		t.Fatalf("unexpected initial share result: %+v", first)
	}
	if err := db.Model(&model.UserCompetitiveStats{}).Where("user_id = ? AND game_type = ?", 1, 1).Update("highest_break", 93).Error; err != nil {
		t.Fatalf("update current profile: %v", err)
	}
	second := loadShareForCurrentProfileTest(t, svcCtx)
	if second.MyScore != 60 || second.OpponentScore != 40 {
		t.Fatalf("immutable share score changed: %+v", second)
	}
	if summaryValue(second.SummaryStats, "我的最高单杆") != "93 分" {
		t.Fatalf("share must use the latest competitive profile: %+v", second.SummaryStats)
	}
}

func TestMatchShareDataUsesFactsWhenReadModelIsDisabledOrIncomplete(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		readMode string
	}{
		{name: "disabled", readMode: "disabled"},
		{name: "enabled with missing game snapshots", readMode: "enabled"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
			if err != nil {
				t.Fatalf("open sqlite: %v", err)
			}
			if err := db.AutoMigrate(&model.User{}, &model.Opponent{}, &model.Match{}, &model.MatchRound{}, &model.MatchAction{}, &model.MatchAchievement{}, &model.UserRanking{}, &model.RankChangeLog{}, &model.UserCompetitiveStats{}); err != nil {
				t.Fatalf("prepare fallback share schema: %v", err)
			}
			svcCtx := &svc.ServiceContext{
				DB:                   db,
				UserModel:            model.NewUserModel(db),
				MatchModel:           model.NewMatchModel(db),
				RankingModel:         model.NewRankingModel(db),
				CompetitiveReadModel: model.NewCompetitiveReadModel(db),
				Config:               config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: testCase.readMode}},
			}
			if err := db.Create(&[]model.User{{Id: 1, Nickname: "甲"}, {Id: 2, Nickname: "乙"}}).Error; err != nil {
				t.Fatalf("seed fallback share users: %v", err)
			}
			opponentID := int64(2)
			win, loss := 1, 2
			completedAt := time.Date(2026, 8, 12, 21, 0, 0, 0, time.UTC)
			if err := db.Create(&[]model.Match{
				{Id: 40, UserId: 1, OpponentId: &opponentID, OpponentName: "乙", GameType: 1, MatchMode: model.MatchModeRanked, Status: 2, Result: &win, MyScore: 60, OpponentScore: 40, MatchTime: completedAt, CompletedAt: &completedAt},
				{Id: 41, UserId: 1, OpponentName: "访客甲", GameType: 1, MatchMode: model.MatchModeRanked, Status: 2, Result: &loss, MyScore: 80, OpponentScore: 90, MatchTime: completedAt.Add(time.Hour), CompletedAt: &completedAt},
				{Id: 42, UserId: 2, OpponentName: "访客乙", GameType: 1, MatchMode: model.MatchModeRanked, Status: 2, Result: &win, MyScore: 70, OpponentScore: 30, MatchTime: completedAt.Add(2 * time.Hour), CompletedAt: &completedAt},
			}).Error; err != nil {
				t.Fatalf("seed fallback share matches: %v", err)
			}
			if err := db.Create(&[]model.MatchAction{
				{MatchId: 40, RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 40},
				{MatchId: 41, RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 80},
				{MatchId: 42, RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 70},
			}).Error; err != nil {
				t.Fatalf("seed fallback share actions: %v", err)
			}
			snapshots := []model.UserCompetitiveStats{
				{UserId: 1, GameType: 0, TotalMatches: 10, Wins: 9, Revision: 99},
				{UserId: 2, GameType: 0, TotalMatches: 10, Wins: 8, Revision: 99},
			}
			if testCase.readMode == "disabled" {
				snapshots = append(snapshots,
					model.UserCompetitiveStats{UserId: 1, GameType: 1, HighestBreak: 147},
					model.UserCompetitiveStats{UserId: 2, GameType: 1, HighestBreak: 130},
				)
			}
			if err := db.Create(&snapshots).Error; err != nil {
				t.Fatalf("seed fallback share snapshots: %v", err)
			}
			if err := db.Create(&[]model.UserRanking{
				{UserId: 1, GameType: 1, TotalWins: 1, TotalLosses: 1}, {UserId: 1, GameType: 2}, {UserId: 1, GameType: 3}, {UserId: 1, GameType: 4},
				{UserId: 2, GameType: 1, TotalWins: 1, TotalLosses: 1}, {UserId: 2, GameType: 2}, {UserId: 2, GameType: 3}, {UserId: 2, GameType: 4},
			}).Error; err != nil {
				t.Fatalf("seed fallback share rankings: %v", err)
			}

			resp, err := NewGetMatchShareDataLogic(context.WithValue(context.Background(), "user_id", int64(1)), svcCtx).GetMatchShareData(&types.GetMatchShareDataReq{MatchId: 40})
			if err != nil || !resp.Success || resp.Data == nil {
				t.Fatalf("get fallback share: resp=%#v err=%v", resp, err)
			}
			if summaryValue(resp.Data.SummaryStats, "我的胜率") != "50%" || summaryValue(resp.Data.SummaryStats, "对手胜率") != "50%" || summaryValue(resp.Data.SummaryStats, "我的最高单杆") != "80 分" || summaryValue(resp.Data.SummaryStats, "对手最高单杆") != "70 分" {
				t.Fatalf("share must use facts instead of unavailable snapshots: %+v", resp.Data.SummaryStats)
			}
		})
	}
}

func loadShareForCurrentProfileTest(t *testing.T, svcCtx *svc.ServiceContext) *types.MatchShareData {
	t.Helper()
	resp, err := NewGetMatchShareDataLogic(context.WithValue(context.Background(), "user_id", int64(1)), svcCtx).GetMatchShareData(&types.GetMatchShareDataReq{MatchId: 21})
	if err != nil || !resp.Success || resp.Data == nil {
		t.Fatalf("get match share data: resp=%#v err=%v", resp, err)
	}
	return resp.Data
}

func summaryValue(items []types.MatchSummaryItem, label string) string {
	for _, item := range items {
		if item.Label == label {
			return item.Value
		}
	}
	return ""
}
