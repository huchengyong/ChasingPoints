package match

import (
	"context"
	"math"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetMatchDetailKeepsCoreScoreAndUsesLatestCompetitiveProfiles(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}, &model.MatchRound{}, &model.MatchAction{}, &model.MatchAchievement{}, &model.UserCompetitiveStats{}); err != nil {
		t.Fatalf("prepare detail schema: %v", err)
	}
	miniRedis := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	svcCtx := &svc.ServiceContext{
		DB:                   db,
		Redis:                redisClient,
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
	completedAt := time.Date(2026, 8, 12, 18, 0, 0, 0, time.UTC)
	if err := db.Create(&model.Match{
		Id: 20, UserId: 1, OpponentId: &opponentID, OpponentName: "乙", GameType: 3,
		Status: 2, Result: &result, MyScore: 3, OpponentScore: 1, MatchTime: completedAt, CompletedAt: &completedAt, AchievementSyncedAt: &completedAt,
	}).Error; err != nil {
		t.Fatalf("seed match: %v", err)
	}
	seedCurrentProfiles(t, db, 5, 10, 8, 4, 10, 7)

	first := loadDetailForProfileTest(t, svcCtx)
	if first.MyScore != 3 || first.OpponentScore != 1 || first.MyWinRate != 50 || first.OpponentWinRate != 40 || first.MyMaxScore != 8 || first.OpponentMaxScore != 7 {
		t.Fatalf("unexpected initial detail: %+v", first)
	}

	if err := db.Model(&model.UserCompetitiveStats{}).Where("user_id = ? AND game_type = ?", 1, 0).Updates(map[string]interface{}{"total_matches": 11, "wins": 6, "revision": 2}).Error; err != nil {
		t.Fatalf("update overall profile: %v", err)
	}
	if err := db.Model(&model.UserCompetitiveStats{}).Where("user_id = ? AND game_type = ?", 1, 3).Update("highest_score", 11).Error; err != nil {
		t.Fatalf("update game profile: %v", err)
	}

	second := loadDetailForProfileTest(t, svcCtx)
	if second.MyScore != 3 || second.OpponentScore != 1 {
		t.Fatalf("immutable match score changed after a later match: %+v", second)
	}
	if math.Abs(second.MyWinRate-(float64(6)/11*100)) > 0.00001 || second.MyMaxScore != 11 {
		t.Fatalf("detail must use latest current profile after a later match: %+v", second)
	}
}

func TestGetMatchDetailUsesFactsWhenReadModelIsDisabledOrIncomplete(t *testing.T) {
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
				t.Fatalf("prepare fallback detail schema: %v", err)
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
				t.Fatalf("seed fallback users: %v", err)
			}
			opponentID := int64(2)
			win, loss := 1, 2
			completedAt := time.Date(2026, 8, 12, 20, 0, 0, 0, time.UTC)
			if err := db.Create(&[]model.Match{
				{Id: 30, UserId: 1, OpponentId: &opponentID, OpponentName: "乙", GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &win, MyScore: 8, OpponentScore: 5, MatchTime: completedAt, CompletedAt: &completedAt},
				{Id: 31, UserId: 1, OpponentName: "访客甲", GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &loss, MyScore: 12, OpponentScore: 13, MatchTime: completedAt.Add(time.Hour), CompletedAt: &completedAt},
				{Id: 32, UserId: 2, OpponentName: "访客乙", GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &win, MyScore: 9, OpponentScore: 4, MatchTime: completedAt.Add(2 * time.Hour), CompletedAt: &completedAt},
			}).Error; err != nil {
				t.Fatalf("seed fallback matches: %v", err)
			}
			snapshots := []model.UserCompetitiveStats{
				{UserId: 1, GameType: 0, TotalMatches: 10, Wins: 9, Revision: 99},
				{UserId: 2, GameType: 0, TotalMatches: 10, Wins: 8, Revision: 99},
			}
			if testCase.readMode == "disabled" {
				snapshots = append(snapshots,
					model.UserCompetitiveStats{UserId: 1, GameType: 3, HighestScore: 99},
					model.UserCompetitiveStats{UserId: 2, GameType: 3, HighestScore: 88},
				)
			}
			if err := db.Create(&snapshots).Error; err != nil {
				t.Fatalf("seed fallback snapshots: %v", err)
			}
			if err := db.Create(&[]model.UserRanking{
				{UserId: 1, GameType: 1}, {UserId: 1, GameType: 2}, {UserId: 1, GameType: 3, TotalWins: 1, TotalLosses: 1}, {UserId: 1, GameType: 4},
				{UserId: 2, GameType: 1}, {UserId: 2, GameType: 2}, {UserId: 2, GameType: 3, TotalWins: 1, TotalLosses: 1}, {UserId: 2, GameType: 4},
			}).Error; err != nil {
				t.Fatalf("seed fallback rankings: %v", err)
			}

			resp, err := NewGetMatchDetailLogic(context.WithValue(context.Background(), "user_id", int64(1)), svcCtx).GetMatchDetail(&types.GetMatchDetailReq{MatchId: 30})
			if err != nil || !resp.Success {
				t.Fatalf("get fallback detail: resp=%#v err=%v", resp, err)
			}
			if resp.Match.MyWinRate != 50 || resp.Match.OpponentWinRate != 50 || resp.Match.MyMaxScore != 12 || resp.Match.OpponentMaxScore != 9 {
				t.Fatalf("detail must use facts instead of unavailable snapshots: %+v", resp.Match)
			}
		})
	}
}

func seedCurrentProfiles(t *testing.T, db *gorm.DB, player1Wins, player1Total, player1High, player2Wins, player2Total, player2High int) {
	t.Helper()
	if err := db.Create(&[]model.UserCompetitiveStats{
		{UserId: 1, GameType: 0, TotalMatches: player1Total, Wins: player1Wins, Revision: 1},
		{UserId: 1, GameType: 3, HighestScore: player1High},
		{UserId: 2, GameType: 0, TotalMatches: player2Total, Wins: player2Wins, Revision: 1},
		{UserId: 2, GameType: 3, HighestScore: player2High},
	}).Error; err != nil {
		t.Fatalf("seed current profiles: %v", err)
	}
}

func loadDetailForProfileTest(t *testing.T, svcCtx *svc.ServiceContext) types.MatchDetailData {
	t.Helper()
	resp, err := NewGetMatchDetailLogic(context.WithValue(context.Background(), "user_id", int64(1)), svcCtx).GetMatchDetail(&types.GetMatchDetailReq{MatchId: 20})
	if err != nil || !resp.Success {
		t.Fatalf("get match detail: resp=%#v err=%v", resp, err)
	}
	return resp.Match
}
