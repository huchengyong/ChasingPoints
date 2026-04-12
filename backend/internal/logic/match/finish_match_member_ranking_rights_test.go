package match

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newFinishMatchRightsTestSvc(t *testing.T) *svc.ServiceContext {
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
		&model.AchievementRewardConfig{},
		&model.MemberGrowthProfile{},
		&model.MemberGrowthLog{},
	); err != nil {
		t.Fatalf("prepare finish rights schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:                       db,
		UserModel:                model.NewUserModel(db),
		MatchModel:               model.NewMatchModel(db),
		RankingModel:             model.NewRankingModel(db),
		NotificationModel:        model.NewNotificationModel(db),
		MemberGrowthProfileModel: model.NewMemberGrowthProfileModel(db),
		MemberGrowthLogModel:     model.NewMemberGrowthLogModel(db),
	}
}

func finishRightsCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func seedFinishRightsUser(t *testing.T, svcCtx *svc.ServiceContext, user model.User) {
	t.Helper()
	if err := svcCtx.UserModel.Create(&user); err != nil {
		t.Fatalf("create user %d: %v", user.Id, err)
	}
}

func seedAchievementRewardConfig(t *testing.T, svcCtx *svc.ServiceContext, cfg model.AchievementRewardConfig) {
	t.Helper()
	if err := svcCtx.DB.Create(&cfg).Error; err != nil {
		t.Fatalf("create achievement reward config: %v", err)
	}
}

func seedCompletedRound(t *testing.T, svcCtx *svc.ServiceContext, round model.MatchRound) {
	t.Helper()
	if err := svcCtx.DB.Create(&round).Error; err != nil {
		t.Fatalf("create round: %v", err)
	}
}

func TestFinishMatchOrdinaryUserDoesNotGetAchievementRankingScore(t *testing.T) {
	svcCtx := newFinishMatchRightsTestSvc(t)
	now := time.Date(2026, 4, 12, 10, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	seedFinishRightsUser(t, svcCtx, model.User{Id: 1001, Nickname: "普通玩家"})
	seedAchievementRewardConfig(t, svcCtx, model.AchievementRewardConfig{GameType: 3, AchievementType: "break_and_run", Name: "炸清", RewardScore: 6})

	if err := svcCtx.MatchModel.Create(&model.Match{
		Id:            401,
		UserId:        1001,
		OpponentName:  "路人对手",
		GameType:      3,
		MyScore:       1,
		OpponentScore: 0,
		Status:        1,
		MatchTime:     now,
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}
	winner := 1
	seedCompletedRound(t, svcCtx, model.MatchRound{
		MatchId:       401,
		RoundNo:       1,
		MyScore:       1,
		OpponentScore: 0,
		Winner:        &winner,
		WinType:       "break_clear",
	})

	resp, err := NewFinishMatchLogic(finishRightsCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId:        401,
		ClientActionId: "finish-rights-401",
		BaseRevision:   0,
	})
	if err != nil {
		t.Fatalf("finish match: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %#v", resp)
	}

	log, err := svcCtx.RankingModel.FindMatchRankChangeByUserAndGameType(401, 1001, 3)
	if err != nil || log == nil {
		t.Fatalf("find player1 rank log: %v %+v", err, log)
	}
	if log.AchievementScore != 0 || log.FinalChange != 20 {
		t.Fatalf("expected ordinary user base-only rank gain, got %+v", log)
	}
}

func TestFinishMatchMemberGetsAchievementRankingScoreByLevelMultiplier(t *testing.T) {
	svcCtx := newFinishMatchRightsTestSvc(t)
	now := time.Date(2026, 4, 12, 11, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	expiresAt := now.Add(24 * time.Hour)
	seedFinishRightsUser(t, svcCtx, model.User{Id: 1001, Nickname: "会员玩家", MemberExpiresAt: &expiresAt})
	if err := svcCtx.MemberGrowthProfileModel.Create(&model.MemberGrowthProfile{UserId: 1001, GrowthPoints: 78, GrowthLevel: 3}); err != nil {
		t.Fatalf("create growth profile: %v", err)
	}
	seedAchievementRewardConfig(t, svcCtx, model.AchievementRewardConfig{GameType: 3, AchievementType: "break_and_run", Name: "炸清", RewardScore: 6})

	if err := svcCtx.MatchModel.Create(&model.Match{
		Id:            402,
		UserId:        1001,
		OpponentName:  "路人对手",
		GameType:      3,
		MyScore:       1,
		OpponentScore: 0,
		Status:        1,
		MatchTime:     now,
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}
	winner := 1
	seedCompletedRound(t, svcCtx, model.MatchRound{
		MatchId:       402,
		RoundNo:       1,
		MyScore:       1,
		OpponentScore: 0,
		Winner:        &winner,
		WinType:       "break_clear",
	})

	resp, err := NewFinishMatchLogic(finishRightsCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId:        402,
		ClientActionId: "finish-rights-402",
		BaseRevision:   0,
	})
	if err != nil {
		t.Fatalf("finish match: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %#v", resp)
	}

	log, err := svcCtx.RankingModel.FindMatchRankChangeByUserAndGameType(402, 1001, 3)
	if err != nil || log == nil {
		t.Fatalf("find player1 rank log: %v %+v", err, log)
	}
	if log.AchievementScore != 7 || log.FinalChange != 27 {
		t.Fatalf("expected lv3 member to get 7 achievement score and 27 final change, got %+v", log)
	}
}

func TestFinishMatchMemberAchievementRankingScoreHonorsDailyCap(t *testing.T) {
	svcCtx := newFinishMatchRightsTestSvc(t)
	now := time.Date(2026, 4, 12, 12, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	expiresAt := now.Add(24 * time.Hour)
	seedFinishRightsUser(t, svcCtx, model.User{Id: 1001, Nickname: "会员玩家", MemberExpiresAt: &expiresAt})
	if err := svcCtx.MemberGrowthProfileModel.Create(&model.MemberGrowthProfile{UserId: 1001, GrowthPoints: 760, GrowthLevel: 5}); err != nil {
		t.Fatalf("create growth profile: %v", err)
	}
	seedAchievementRewardConfig(t, svcCtx, model.AchievementRewardConfig{GameType: 1, AchievementType: "break_147", Name: "单杆147", RewardScore: 30})

	if err := svcCtx.DB.Create(&model.RankChangeLog{
		UserId:           1001,
		MatchId:          99,
		ChangeType:       "match_result",
		GameType:         1,
		Result:           "win",
		BaseScore:        20,
		AchievementScore: 195,
		FinalChange:      215,
		BeforeScore:      100,
		AfterScore:       315,
		BeforeLevel:      1,
		AfterLevel:       1,
		Remark:           "{}",
		EffectiveAt:      now.Add(-time.Hour),
	}).Error; err != nil {
		t.Fatalf("seed prior rank log: %v", err)
	}

	if err := svcCtx.MatchModel.Create(&model.Match{
		Id:            403,
		UserId:        1001,
		OpponentName:  "路人对手",
		GameType:      1,
		MyScore:       2,
		OpponentScore: 1,
		Status:        1,
		MatchTime:     now,
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}
	winner := 1
	seedCompletedRound(t, svcCtx, model.MatchRound{
		MatchId:       403,
		RoundNo:       1,
		MyScore:       2,
		OpponentScore: 1,
		Winner:        &winner,
		WinType:       "normal",
	})
	seedCompletedRound(t, svcCtx, model.MatchRound{
		MatchId:       403,
		RoundNo:       2,
		MyScore:       1,
		OpponentScore: 0,
		Winner:        &winner,
		WinType:       "normal",
	})
	if err := svcCtx.DB.Create(&model.MatchAction{
		MatchId:     403,
		RoundNo:     2,
		ActionType:  "score",
		Actor:       1,
		ScoreChange: 147,
	}).Error; err != nil {
		t.Fatalf("seed snooker action: %v", err)
	}

	resp, err := NewFinishMatchLogic(finishRightsCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId:        403,
		ClientActionId: "finish-rights-403",
		BaseRevision:   0,
	})
	if err != nil {
		t.Fatalf("finish match: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %#v", resp)
	}

	log, err := svcCtx.RankingModel.FindMatchRankChangeByUserAndGameType(403, 1001, 1)
	if err != nil || log == nil {
		t.Fatalf("find player1 rank log: %v %+v", err, log)
	}
	if log.AchievementScore != 5 || log.FinalChange != 25 {
		t.Fatalf("expected daily cap to clamp achievement score to 5 and final change to 25, got %+v", log)
	}
}
