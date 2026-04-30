package user

import (
	"context"
	"reflect"
	"testing"
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newUserReputationTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.ReputationConfig{}, &model.UserReputationProfile{}, &model.UserReputationLog{}); err != nil {
		t.Fatalf("prepare user reputation schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:                         db,
		ReputationConfigModel:      model.NewReputationConfigModel(db),
		UserReputationProfileModel: model.NewUserReputationProfileModel(db),
		UserReputationLogModel:     model.NewUserReputationLogModel(db),
	}
}

func newUserContext(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func mustSeedUserReputationConfig(t *testing.T, svcCtx *svc.ServiceContext, cfg *model.ReputationConfig) {
	t.Helper()
	if err := svcCtx.ReputationConfigModel.Upsert(cfg); err != nil {
		t.Fatalf("seed reputation config: %v", err)
	}
}

func mustSeedUserReputationProfile(t *testing.T, svcCtx *svc.ServiceContext, profile *model.UserReputationProfile) {
	t.Helper()
	if err := svcCtx.UserReputationProfileModel.Save(profile); err != nil {
		t.Fatalf("seed reputation profile: %v", err)
	}
}

func mustSeedUserReputationLog(t *testing.T, svcCtx *svc.ServiceContext, log *model.UserReputationLog) {
	t.Helper()
	if err := svcCtx.UserReputationLogModel.Create(log); err != nil {
		t.Fatalf("seed reputation log: %v", err)
	}
}

func TestGetUserReputationReturnsInitialScoreAndGoodWithoutProfile(t *testing.T) {
	svcCtx := newUserReputationTestSvc(t)
	cfg := model.DefaultReputationConfig()
	cfg.SetBaseRules(model.ReputationBaseRules{
		MaxScore:         100,
		InitialScore:     88,
		BanThreshold:     60,
		BanDurationHours: 24,
		MinScore:         0,
	})
	mustSeedUserReputationConfig(t, svcCtx, cfg)

	resp, err := NewGetUserReputationLogic(newUserContext(2001), svcCtx).GetUserReputation()
	if err != nil {
		t.Fatalf("get user reputation: %v", err)
	}
	if !resp.Success || resp.Score != 88 {
		t.Fatalf("expected initial score fallback, got %#v", resp)
	}
	if resp.Status != "good" || resp.StatusText != "良好" {
		t.Fatalf("expected good reputation status, got %#v", resp)
	}
	if resp.BanUntil != "" {
		t.Fatalf("expected no ban_until for healthy user, got %#v", resp)
	}

	profile, err := svcCtx.UserReputationProfileModel.FindByUserID(2001)
	if err != nil {
		t.Fatalf("find profile after read-only get user reputation: %v", err)
	}
	if profile != nil {
		t.Fatalf("expected read-only path to avoid creating profile, got %#v", profile)
	}
}

func TestGetUserReputationReturnsSuccessFalseWhenUserIDMissing(t *testing.T) {
	svcCtx := newUserReputationTestSvc(t)

	resp, err := NewGetUserReputationLogic(context.Background(), svcCtx).GetUserReputation()
	if err != nil {
		t.Fatalf("get user reputation with missing user_id: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected failure response when user_id missing, got %#v", resp)
	}
}

func TestGetUserReputationReturnsSuccessFalseWhenReceiverNil(t *testing.T) {
	var logic *GetUserReputationLogic

	resp, err := logic.GetUserReputation()
	if err != nil {
		t.Fatalf("get user reputation with nil receiver: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected failure response when receiver nil, got %#v", resp)
	}
}

func TestGetUserReputationReturnsSuccessFalseWhenProfileLookupFails(t *testing.T) {
	resp, err := NewGetUserReputationLogic(newUserContext(2005), &svc.ServiceContext{}).GetUserReputation()
	if err != nil {
		t.Fatalf("get user reputation with nil dependencies: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected failure response when profile lookup fails, got %#v", resp)
	}
}

func TestGetUserReputationReturnsBanUntilWhenRestricted(t *testing.T) {
	svcCtx := newUserReputationTestSvc(t)
	banUntil := logicx.NowUTC8().Add(24 * time.Hour)

	mustSeedUserReputationProfile(t, svcCtx, &model.UserReputationProfile{
		UserID:          2002,
		ReputationScore: 40,
		BanUntil:        &banUntil,
	})

	resp, err := NewGetUserReputationLogic(newUserContext(2002), svcCtx).GetUserReputation()
	if err != nil {
		t.Fatalf("get user reputation: %v", err)
	}
	if !resp.Success || resp.Status != "restricted" {
		t.Fatalf("expected restricted reputation status, got %#v", resp)
	}
	if resp.StatusText != "禁赛" {
		t.Fatalf("expected restricted status text to be 禁赛, got %#v", resp)
	}
	if resp.BanUntil == "" {
		t.Fatalf("expected non-empty ban_until, got %#v", resp)
	}
}

func TestGetUserReputationReturnsRecoveredDisplayScoreWithoutPersistingProfile(t *testing.T) {
	svcCtx := newUserReputationTestSvc(t)
	cfg := model.DefaultReputationConfig()
	cfg.SetBaseRules(model.ReputationBaseRules{
		MaxScore:         100,
		InitialScore:     88,
		BanThreshold:     60,
		BanDurationHours: 24,
		MinScore:         0,
	})
	cfg.SetRecoveryRules(model.ReputationRecoveryRules{
		Enabled:         true,
		RecoverPerHour:  5,
		RecoverMaxScore: 95,
	})
	mustSeedUserReputationConfig(t, svcCtx, cfg)

	lastRecoveredAt := logicx.NowUTC8().Add(-(2 * time.Hour) - (10 * time.Minute))
	mustSeedUserReputationProfile(t, svcCtx, &model.UserReputationProfile{
		UserID:          2010,
		ReputationScore: 70,
		LastRecoveredAt: &lastRecoveredAt,
	})

	resp, err := NewGetUserReputationLogic(newUserContext(2010), svcCtx).GetUserReputation()
	if err != nil {
		t.Fatalf("get user reputation with recovery preview: %v", err)
	}
	if !resp.Success || resp.Score != 80 {
		t.Fatalf("expected recovered display score 80, got %#v", resp)
	}

	profile, err := svcCtx.UserReputationProfileModel.FindByUserID(2010)
	if err != nil {
		t.Fatalf("reload profile after read-only recovery: %v", err)
	}
	if profile == nil {
		t.Fatalf("expected stored profile to still exist")
	}
	if profile.ReputationScore != 70 {
		t.Fatalf("expected DB profile score to remain 70, got %#v", profile)
	}
}

func TestGetUserReputationLogsReturnsOnlyCurrentUserWithoutRawDetail(t *testing.T) {
	svcCtx := newUserReputationTestSvc(t)
	matchID := int64(9901)

	mustSeedUserReputationLog(t, svcCtx, &model.UserReputationLog{
		UserID:          2003,
		MatchID:         &matchID,
		ChangeType:      model.ReputationChangeTypePenalty,
		ChangeScore:     -10,
		BeforeScore:     90,
		AfterScore:      80,
		ReasonCode:      model.ReputationReasonDurationAbnormal,
		ReasonDetail:    "中式八球 10 局总时长 5 分钟",
		OperatorAdminID: 0,
	})
	mustSeedUserReputationLog(t, svcCtx, &model.UserReputationLog{
		UserID:          2004,
		ChangeType:      model.ReputationChangeTypePenalty,
		ChangeScore:     -8,
		BeforeScore:     88,
		AfterScore:      80,
		ReasonCode:      model.ReputationReasonSameOpponentHighFrequency,
		ReasonDetail:    "30 分钟内与同一对手完成 6 场",
		OperatorAdminID: 0,
	})

	resp, err := NewGetUserReputationLogsLogic(newUserContext(2003), svcCtx).GetUserReputationLogs(&types.UserReputationLogsReq{
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("get user reputation logs: %v", err)
	}
	if !resp.Success || resp.Total != 1 || len(resp.List) != 1 {
		t.Fatalf("expected current user logs only, got %#v", resp)
	}

	item := resp.List[0]
	if item.MatchId != matchID || item.ReasonCode != model.ReputationReasonDurationAbnormal {
		t.Fatalf("expected current user duration abnormal log, got %#v", item)
	}
	if item.ChangeTypeText != "扣分" || item.ReasonText != "时长异常" {
		t.Fatalf("expected display texts, got %#v", item)
	}
	if _, ok := reflect.TypeOf(types.UserReputationLogItem{}).FieldByName("ReasonDetail"); ok {
		t.Fatalf("user reputation log item should not expose raw reason detail")
	}
}

func TestGetUserReputationLogsReturnsSuccessFalseWhenUserIDMissing(t *testing.T) {
	svcCtx := newUserReputationTestSvc(t)

	resp, err := NewGetUserReputationLogsLogic(context.Background(), svcCtx).GetUserReputationLogs(&types.UserReputationLogsReq{
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("get user reputation logs with missing user_id: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected failure response when user_id missing, got %#v", resp)
	}
}

func TestGetUserReputationLogsReturnsSuccessFalseWhenReceiverNil(t *testing.T) {
	var logic *GetUserReputationLogsLogic

	resp, err := logic.GetUserReputationLogs(&types.UserReputationLogsReq{
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("get user reputation logs with nil receiver: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected failure response when receiver nil, got %#v", resp)
	}
}

func TestGetUserReputationLogsReturnsSuccessFalseWhenQueryFails(t *testing.T) {
	resp, err := NewGetUserReputationLogsLogic(newUserContext(2006), &svc.ServiceContext{}).GetUserReputationLogs(&types.UserReputationLogsReq{
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("get user reputation logs with nil model: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected failure response when log query fails, got %#v", resp)
	}
}
