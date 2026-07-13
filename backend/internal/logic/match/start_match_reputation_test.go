package match

import (
	"context"
	"strings"
	"testing"
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newStartMatchReputationTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Match{},
		&model.Opponent{},
		&model.ReputationConfig{},
		&model.UserReputationProfile{},
	); err != nil {
		t.Fatalf("prepare start reputation schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:                         db,
		UserModel:                  model.NewUserModel(db),
		MatchModel:                 model.NewMatchModel(db),
		ReputationConfigModel:      model.NewReputationConfigModel(db),
		UserReputationProfileModel: model.NewUserReputationProfileModel(db),
	}
}

func startReputationCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func seedStartReputationUsers(t *testing.T, svcCtx *svc.ServiceContext, users ...model.User) {
	t.Helper()
	for _, user := range users {
		u := user
		if err := svcCtx.UserModel.Create(&u); err != nil {
			t.Fatalf("create user %d: %v", u.Id, err)
		}
	}
}

func TestStartMatchBlocksWhenUserIsInReputationBan(t *testing.T) {
	svcCtx := newStartMatchReputationTestSvc(t)
	now := logicx.NowUTC8()
	banUntil := now.Add(2 * time.Hour)
	seedStartReputationUsers(t, svcCtx,
		model.User{Id: 1001, Nickname: "发起人"},
		model.User{Id: 2002, Nickname: "对手"},
	)
	if err := svcCtx.UserReputationProfileModel.Save(&model.UserReputationProfile{
		UserID:          1001,
		ReputationScore: 55,
		BanUntil:        &banUntil,
		LastRecoveredAt: &now,
	}); err != nil {
		t.Fatalf("seed reputation profile: %v", err)
	}

	resp, err := NewStartMatchLogic(startReputationCtx(1001), svcCtx).StartMatch(&types.StartMatchReq{
		GameType:      3,
		OpponentId:    2002,
		OpponentName:  "对手",
		OpponentAvatar: "",
	})
	if err != nil {
		t.Fatalf("start match: %v", err)
	}
	if !resp.Success || resp.Action != startMatchActionBlocked || resp.BlockReason != startMatchBlockReasonLowReputationBan {
		t.Fatalf("expected low reputation block, got %#v", resp)
	}
	if !strings.Contains(resp.Message, logicx.FormatUTC8Time(banUntil)) {
		t.Fatalf("expected ban message to include ban time, got %q", resp.Message)
	}
}

func TestStartMatchRecoversScoreButStillBlocksBeforeBanEnds(t *testing.T) {
	svcCtx := newStartMatchReputationTestSvc(t)
	now := logicx.NowUTC8()
	lastRecoveredAt := now.Add(-(2*time.Hour + 10*time.Minute))
	banUntil := now.Add(30 * time.Minute)

	cfg := model.DefaultReputationConfig()
	cfg.SetRecoveryRules(model.ReputationRecoveryRules{
		Enabled:         true,
		RecoverPerHour:  5,
		RecoverMaxScore: 100,
	})
	if err := svcCtx.ReputationConfigModel.Upsert(cfg); err != nil {
		t.Fatalf("seed reputation config: %v", err)
	}
	seedStartReputationUsers(t, svcCtx,
		model.User{Id: 1001, Nickname: "发起人"},
		model.User{Id: 2002, Nickname: "对手"},
	)
	if err := svcCtx.UserReputationProfileModel.Save(&model.UserReputationProfile{
		UserID:          1001,
		ReputationScore: 50,
		BanUntil:        &banUntil,
		LastRecoveredAt: &lastRecoveredAt,
	}); err != nil {
		t.Fatalf("seed reputation profile: %v", err)
	}

	resp, err := NewStartMatchLogic(startReputationCtx(1001), svcCtx).StartMatch(&types.StartMatchReq{
		GameType:      3,
		OpponentId:    2002,
		OpponentName:  "对手",
		OpponentAvatar: "",
	})
	if err != nil {
		t.Fatalf("start match: %v", err)
	}
	if !resp.Success || resp.Action != startMatchActionBlocked || resp.BlockReason != startMatchBlockReasonLowReputationBan {
		t.Fatalf("expected low reputation block, got %#v", resp)
	}

	profile, err := svcCtx.UserReputationProfileModel.FindByUserID(1001)
	if err != nil {
		t.Fatalf("reload profile: %v", err)
	}
	if profile == nil || profile.ReputationScore != 60 {
		t.Fatalf("expected recovered score 60 before block, got %+v", profile)
	}
}

func TestStartMatchAllowsWhenBanExpired(t *testing.T) {
	svcCtx := newStartMatchReputationTestSvc(t)
	now := logicx.NowUTC8()
	lastRecoveredAt := now.Add(-(2*time.Hour + 10*time.Minute))
	banUntil := now.Add(-5 * time.Minute)

	cfg := model.DefaultReputationConfig()
	cfg.SetRecoveryRules(model.ReputationRecoveryRules{
		Enabled:         true,
		RecoverPerHour:  5,
		RecoverMaxScore: 100,
	})
	if err := svcCtx.ReputationConfigModel.Upsert(cfg); err != nil {
		t.Fatalf("seed reputation config: %v", err)
	}
	seedStartReputationUsers(t, svcCtx,
		model.User{Id: 1001, Nickname: "发起人"},
		model.User{Id: 2002, Nickname: "对手"},
	)
	if err := svcCtx.UserReputationProfileModel.Save(&model.UserReputationProfile{
		UserID:          1001,
		ReputationScore: 50,
		BanUntil:        &banUntil,
		LastRecoveredAt: &lastRecoveredAt,
	}); err != nil {
		t.Fatalf("seed reputation profile: %v", err)
	}

	resp, err := NewStartMatchLogic(startReputationCtx(1001), svcCtx).StartMatch(&types.StartMatchReq{
		GameType:      3,
		OpponentId:    2002,
		OpponentName:  "对手",
		OpponentAvatar: "",
	})
	if err != nil {
		t.Fatalf("start match: %v", err)
	}
	if !resp.Success || resp.Action != startMatchActionCreated || resp.MatchId <= 0 {
		t.Fatalf("expected created match after ban expiry, got %#v", resp)
	}

	profile, err := svcCtx.UserReputationProfileModel.FindByUserID(1001)
	if err != nil {
		t.Fatalf("reload profile: %v", err)
	}
	if profile == nil || profile.ReputationScore != 60 {
		t.Fatalf("expected recovered score 60 after successful start, got %+v", profile)
	}
}
