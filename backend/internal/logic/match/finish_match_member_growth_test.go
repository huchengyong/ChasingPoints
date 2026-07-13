package match

import (
	"context"
	"testing"
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newFinishMatchGrowthTestSvc(t *testing.T) *svc.ServiceContext {
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
		&model.MemberGrowthProfile{},
		&model.MemberGrowthLog{},
	); err != nil {
		t.Fatalf("prepare finish growth schema: %v", err)
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

func finishGrowthCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func TestFinishMatchAwardsGrowthForActiveMembers(t *testing.T) {
	svcCtx := newFinishMatchGrowthTestSvc(t)
	now := logicx.NowUTC8()
	expiresAt := now.Add(24 * time.Hour)
	opponentID := int64(2002)

	for _, user := range []model.User{
		{Id: 1001, Nickname: "选手甲", MemberExpiresAt: &expiresAt},
		{Id: opponentID, Nickname: "选手乙", MemberExpiresAt: &expiresAt},
	} {
		u := user
		if err := svcCtx.UserModel.Create(&u); err != nil {
			t.Fatalf("create user %d: %v", u.Id, err)
		}
	}

	if err := svcCtx.MatchModel.Create(&model.Match{
		Id:            301,
		UserId:        1001,
		OpponentId:    &opponentID,
		OpponentName:  "选手乙",
		GameType:      3,
		MyScore:       5,
		OpponentScore: 5,
		Status:        1,
		MatchTime:     now,
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}

	resp, err := NewFinishMatchLogic(finishGrowthCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId:        301,
		ClientActionId: "finish-growth-301",
		BaseRevision:   0,
	})
	if err != nil {
		t.Fatalf("finish match: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected finish success, got %#v", resp)
	}

	profile1, err := svcCtx.MemberGrowthProfileModel.FindByUserId(1001)
	if err != nil {
		t.Fatalf("find growth profile 1001: %v", err)
	}
	if profile1 == nil || profile1.GrowthPoints != 1 || profile1.GrowthLevel != 1 {
		t.Fatalf("unexpected player1 growth profile: %+v", profile1)
	}

	profile2, err := svcCtx.MemberGrowthProfileModel.FindByUserId(opponentID)
	if err != nil {
		t.Fatalf("find growth profile 2002: %v", err)
	}
	if profile2 == nil || profile2.GrowthPoints != 1 || profile2.GrowthLevel != 1 {
		t.Fatalf("unexpected player2 growth profile: %+v", profile2)
	}
}

func TestFinishMatchDoesNotRollbackWhenGrowthInfrastructureMissing(t *testing.T) {
	svcCtx := newFinishMatchGrowthTestSvc(t)
	now := logicx.NowUTC8()
	expiresAt := now.Add(24 * time.Hour)
	opponentID := int64(2002)

	for _, user := range []model.User{
		{Id: 1001, Nickname: "选手甲", MemberExpiresAt: &expiresAt},
		{Id: opponentID, Nickname: "选手乙", MemberExpiresAt: &expiresAt},
	} {
		u := user
		if err := svcCtx.UserModel.Create(&u); err != nil {
			t.Fatalf("create user %d: %v", u.Id, err)
		}
	}

	if err := svcCtx.MatchModel.Create(&model.Match{
		Id:            302,
		UserId:        1001,
		OpponentId:    &opponentID,
		OpponentName:  "选手乙",
		GameType:      3,
		MyScore:       4,
		OpponentScore: 4,
		Status:        1,
		MatchTime:     now,
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}

	svcCtx.MemberGrowthProfileModel = nil
	svcCtx.MemberGrowthLogModel = nil

	resp, err := NewFinishMatchLogic(finishGrowthCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId:        302,
		ClientActionId: "finish-growth-302",
		BaseRevision:   0,
	})
	if err != nil {
		t.Fatalf("finish match: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected finish success even when growth fails, got %#v", resp)
	}

	storedMatch, err := svcCtx.MatchModel.FindById(302)
	if err != nil || storedMatch == nil {
		t.Fatalf("reload match: %v %+v", err, storedMatch)
	}
	if storedMatch.Status != 2 {
		t.Fatalf("expected match status finished, got %+v", storedMatch)
	}
}

func TestFinishMatchReplayCanCompensateGrowthAfterTransientFailure(t *testing.T) {
	svcCtx := newFinishMatchGrowthTestSvc(t)
	now := logicx.NowUTC8()
	expiresAt := now.Add(24 * time.Hour)
	opponentID := int64(2002)

	for _, user := range []model.User{
		{Id: 1001, Nickname: "选手甲", MemberExpiresAt: &expiresAt},
		{Id: opponentID, Nickname: "选手乙", MemberExpiresAt: &expiresAt},
	} {
		u := user
		if err := svcCtx.UserModel.Create(&u); err != nil {
			t.Fatalf("create user %d: %v", u.Id, err)
		}
	}

	if err := svcCtx.MatchModel.Create(&model.Match{
		Id:            303,
		UserId:        1001,
		OpponentId:    &opponentID,
		OpponentName:  "选手乙",
		GameType:      3,
		MyScore:       4,
		OpponentScore: 4,
		Status:        1,
		MatchTime:     now,
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}

	originalProfileModel := svcCtx.MemberGrowthProfileModel
	originalLogModel := svcCtx.MemberGrowthLogModel
	svcCtx.MemberGrowthProfileModel = nil
	svcCtx.MemberGrowthLogModel = nil

	resp, err := NewFinishMatchLogic(finishGrowthCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId:        303,
		ClientActionId: "finish-growth-303",
		BaseRevision:   0,
	})
	if err != nil {
		t.Fatalf("first finish match: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected first finish success, got %#v", resp)
	}

	svcCtx.MemberGrowthProfileModel = originalProfileModel
	svcCtx.MemberGrowthLogModel = originalLogModel

	replayResp, err := NewFinishMatchLogic(finishGrowthCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId:        303,
		ClientActionId: "finish-growth-303",
		BaseRevision:   0,
	})
	if err != nil {
		t.Fatalf("replay finish match: %v", err)
	}
	if !replayResp.Success {
		t.Fatalf("expected replay finish success, got %#v", replayResp)
	}

	profile1, err := svcCtx.MemberGrowthProfileModel.FindByUserId(1001)
	if err != nil || profile1 == nil || profile1.GrowthPoints != 1 {
		t.Fatalf("expected compensated growth for player1, got %+v err=%v", profile1, err)
	}
	profile2, err := svcCtx.MemberGrowthProfileModel.FindByUserId(opponentID)
	if err != nil || profile2 == nil || profile2.GrowthPoints != 1 {
		t.Fatalf("expected compensated growth for player2, got %+v err=%v", profile2, err)
	}
}
