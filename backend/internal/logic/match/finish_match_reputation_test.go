package match

import (
	"context"
	"strings"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newFinishMatchReputationTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Match{},
		&model.MatchRound{},
		&model.MatchAction{},
		&model.UserRanking{},
		&model.RankChangeLog{},
		&model.Notification{},
		&model.ReputationConfig{},
		&model.UserReputationProfile{},
		&model.UserReputationLog{},
	); err != nil {
		t.Fatalf("prepare finish reputation schema: %v", err)
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
		DB:                         db,
		UserModel:                  model.NewUserModel(db),
		MatchModel:                 model.NewMatchModel(db),
		RankingModel:               model.NewRankingModel(db),
		NotificationModel:          model.NewNotificationModel(db),
		ReputationConfigModel:      model.NewReputationConfigModel(db),
		UserReputationProfileModel: model.NewUserReputationProfileModel(db),
		UserReputationLogModel:     model.NewUserReputationLogModel(db),
		Config:                     config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}},
	}
}

func finishReputationCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func seedFinishReputationUsers(t *testing.T, svcCtx *svc.ServiceContext, users ...model.User) {
	t.Helper()
	for _, user := range users {
		u := user
		if err := svcCtx.UserModel.Create(&u); err != nil {
			t.Fatalf("create user %d: %v", u.Id, err)
		}
	}
}

func seedFinishReputationMatch(t *testing.T, svcCtx *svc.ServiceContext, match model.Match, rounds int) {
	t.Helper()
	if err := svcCtx.MatchModel.Create(&match); err != nil {
		t.Fatalf("create match %d: %v", match.Id, err)
	}
	for i := 1; i <= rounds; i++ {
		winner := 1
		if err := svcCtx.DB.Create(&model.MatchRound{
			MatchId:       match.Id,
			RoundNo:       i,
			MyScore:       1,
			OpponentScore: 0,
			Winner:        &winner,
			WinType:       "normal",
		}).Error; err != nil {
			t.Fatalf("create round %d for match %d: %v", i, match.Id, err)
		}
	}
}

func seedHistoricalCompletedMatch(t *testing.T, svcCtx *svc.ServiceContext, match model.Match) {
	t.Helper()
	if err := svcCtx.MatchModel.Create(&match); err != nil {
		t.Fatalf("create historical match %d: %v", match.Id, err)
	}
}

func countUserReputationLogs(t *testing.T, svcCtx *svc.ServiceContext, userID int64, matchID int64) int64 {
	t.Helper()
	var count int64
	if err := svcCtx.DB.Model(&model.UserReputationLog{}).
		Where("user_id = ? AND match_id = ?", userID, matchID).
		Count(&count).Error; err != nil {
		t.Fatalf("count reputation logs: %v", err)
	}
	return count
}

func TestFinishMatchDoesNotPenalizeNormalMatch(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	opponentID := int64(2002)
	now := time.Now().Add(-20 * time.Minute)
	seedFinishReputationUsers(t, svcCtx,
		model.User{Id: 1001, Nickname: "选手甲"},
		model.User{Id: opponentID, Nickname: "选手乙"},
	)
	seedFinishReputationMatch(t, svcCtx, model.Match{
		Id:            501,
		UserId:        1001,
		OpponentId:    &opponentID,
		OpponentName:  "选手乙",
		GameType:      3,
		MyScore:       5,
		OpponentScore: 0,
		Status:        1,
		MatchTime:     now,
	}, 5)

	resp, err := NewFinishMatchLogic(finishReputationCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId:        501,
		ClientActionId: "finish-reputation-normal-501",
		BaseRevision:   0,
	})
	if err != nil {
		t.Fatalf("finish match: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %#v", resp)
	}

	profile1, err := svcCtx.UserReputationProfileModel.FindByUserID(1001)
	if err != nil {
		t.Fatalf("find profile1: %v", err)
	}
	profile2, err := svcCtx.UserReputationProfileModel.FindByUserID(opponentID)
	if err != nil {
		t.Fatalf("find profile2: %v", err)
	}
	if profile1 != nil || profile2 != nil {
		t.Fatalf("expected no reputation profiles for normal match, got %+v %+v", profile1, profile2)
	}
}

func TestFinishMatchPenalizesBothPlayersForDurationAbnormalMatch(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	opponentID := int64(2002)
	now := time.Now().Add(-5 * time.Minute)
	seedFinishReputationUsers(t, svcCtx,
		model.User{Id: 1001, Nickname: "选手甲"},
		model.User{Id: opponentID, Nickname: "选手乙"},
	)
	seedFinishReputationMatch(t, svcCtx, model.Match{
		Id:            502,
		UserId:        1001,
		OpponentId:    &opponentID,
		OpponentName:  "选手乙",
		GameType:      3,
		MyScore:       5,
		OpponentScore: 0,
		Status:        1,
		MatchTime:     now,
	}, 5)

	resp, err := NewFinishMatchLogic(finishReputationCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId:        502,
		ClientActionId: "finish-reputation-duration-502",
		BaseRevision:   0,
	})
	if err != nil {
		t.Fatalf("finish match: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %#v", resp)
	}

	for _, userID := range []int64{1001, opponentID} {
		profile, err := svcCtx.UserReputationProfileModel.FindByUserID(userID)
		if err != nil {
			t.Fatalf("find profile %d: %v", userID, err)
		}
		if profile == nil || profile.ReputationScore != 90 || profile.TotalPenaltyCount != 1 || profile.TotalAbnormalMatchCount != 1 {
			t.Fatalf("unexpected profile for user %d: %+v", userID, profile)
		}
		if countUserReputationLogs(t, svcCtx, userID, 502) != 1 {
			t.Fatalf("expected one reputation log for user %d", userID)
		}
	}
}

func TestFinishMatchPenalizesBothPlayersForSameOpponentHighFrequency(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	opponentID := int64(2002)
	now := time.Now()
	seedFinishReputationUsers(t, svcCtx,
		model.User{Id: 1001, Nickname: "选手甲"},
		model.User{Id: opponentID, Nickname: "选手乙"},
	)
	for i, offset := range []time.Duration{25 * time.Minute, 22 * time.Minute, 18 * time.Minute, 10 * time.Minute} {
		endTime := now.Add(-offset)
		seedHistoricalCompletedMatch(t, svcCtx, model.Match{
			Id:            int64(600 + i),
			UserId:        1001,
			OpponentId:    &opponentID,
			OpponentName:  "选手乙",
			GameType:      3,
			MyScore:       5,
			OpponentScore: 3,
			Status:        2,
			Result:        intPtr(1),
			MatchTime:     endTime.Add(-10 * time.Minute),
			EndTime:       &endTime,
		})
	}
	seedFinishReputationMatch(t, svcCtx, model.Match{
		Id:            503,
		UserId:        1001,
		OpponentId:    &opponentID,
		OpponentName:  "选手乙",
		GameType:      3,
		MyScore:       5,
		OpponentScore: 0,
		Status:        1,
		MatchTime:     now.Add(-20 * time.Minute),
	}, 5)

	resp, err := NewFinishMatchLogic(finishReputationCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId:        503,
		ClientActionId: "finish-reputation-frequency-503",
		BaseRevision:   0,
	})
	if err != nil {
		t.Fatalf("finish match: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %#v", resp)
	}

	for _, userID := range []int64{1001, opponentID} {
		profile, err := svcCtx.UserReputationProfileModel.FindByUserID(userID)
		if err != nil {
			t.Fatalf("find profile %d: %v", userID, err)
		}
		if profile == nil || profile.ReputationScore != 92 || profile.TotalPenaltyCount != 1 || profile.TotalAbnormalMatchCount != 1 {
			t.Fatalf("unexpected frequency penalty profile for user %d: %+v", userID, profile)
		}
	}
}

func TestFinishMatchDoesNotStackPenaltyWhenMultipleRulesHit(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	opponentID := int64(2002)
	now := time.Now()
	seedFinishReputationUsers(t, svcCtx,
		model.User{Id: 1001, Nickname: "选手甲"},
		model.User{Id: opponentID, Nickname: "选手乙"},
	)
	for i, offset := range []time.Duration{25 * time.Minute, 22 * time.Minute, 18 * time.Minute, 10 * time.Minute} {
		endTime := now.Add(-offset)
		seedHistoricalCompletedMatch(t, svcCtx, model.Match{
			Id:            int64(700 + i),
			UserId:        1001,
			OpponentId:    &opponentID,
			OpponentName:  "选手乙",
			GameType:      3,
			MyScore:       5,
			OpponentScore: 3,
			Status:        2,
			Result:        intPtr(1),
			MatchTime:     endTime.Add(-10 * time.Minute),
			EndTime:       &endTime,
		})
	}
	seedFinishReputationMatch(t, svcCtx, model.Match{
		Id:            504,
		UserId:        1001,
		OpponentId:    &opponentID,
		OpponentName:  "选手乙",
		GameType:      3,
		MyScore:       5,
		OpponentScore: 0,
		Status:        1,
		MatchTime:     now.Add(-5 * time.Minute),
	}, 5)

	resp, err := NewFinishMatchLogic(finishReputationCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId:        504,
		ClientActionId: "finish-reputation-combined-504",
		BaseRevision:   0,
	})
	if err != nil {
		t.Fatalf("finish match: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %#v", resp)
	}

	for _, userID := range []int64{1001, opponentID} {
		profile, err := svcCtx.UserReputationProfileModel.FindByUserID(userID)
		if err != nil {
			t.Fatalf("find profile %d: %v", userID, err)
		}
		if profile == nil || profile.ReputationScore != 90 || profile.TotalPenaltyCount != 1 || profile.TotalAbnormalMatchCount != 1 {
			t.Fatalf("unexpected combined penalty profile for user %d: %+v", userID, profile)
		}
		log, err := svcCtx.UserReputationLogModel.FindByUserMatchChangeTypeAndReasonCode(
			userID,
			504,
			model.ReputationChangeTypePenalty,
			matchReputationReasonCodeCombined,
		)
		if err != nil {
			t.Fatalf("find combined log for user %d: %v", userID, err)
		}
		if log == nil || !strings.Contains(log.ReasonDetail, model.ReputationReasonDurationAbnormal) || !strings.Contains(log.ReasonDetail, model.ReputationReasonSameOpponentHighFrequency) {
			t.Fatalf("expected combined reputation log for user %d, got %+v", userID, log)
		}
	}
}

func TestFinishMatchDoesNotRollbackWhenReputationInfrastructureMissing(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	opponentID := int64(2002)
	now := time.Now().Add(-5 * time.Minute)
	seedFinishReputationUsers(t, svcCtx,
		model.User{Id: 1001, Nickname: "选手甲"},
		model.User{Id: opponentID, Nickname: "选手乙"},
	)
	seedFinishReputationMatch(t, svcCtx, model.Match{
		Id:            505,
		UserId:        1001,
		OpponentId:    &opponentID,
		OpponentName:  "选手乙",
		GameType:      3,
		MyScore:       5,
		OpponentScore: 0,
		Status:        1,
		MatchTime:     now,
	}, 5)

	svcCtx.UserReputationProfileModel = nil
	svcCtx.UserReputationLogModel = nil

	resp, err := NewFinishMatchLogic(finishReputationCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId:        505,
		ClientActionId: "finish-reputation-failopen-505",
		BaseRevision:   0,
	})
	if err != nil {
		t.Fatalf("finish match: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected finish success, got %#v", resp)
	}

	storedMatch, err := svcCtx.MatchModel.FindById(505)
	if err != nil || storedMatch == nil || storedMatch.Status != 2 {
		t.Fatalf("expected finished match to persist, got match=%+v err=%v", storedMatch, err)
	}
}

func intPtr(v int) *int {
	return &v
}
