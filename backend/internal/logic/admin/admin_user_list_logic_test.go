package admin

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

func newAdminUserListTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.ReputationConfig{}, &model.UserReputationProfile{}); err != nil {
		t.Fatalf("prepare admin user list schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:                         db,
		UserModel:                  model.NewUserModel(db),
		ReputationConfigModel:      model.NewReputationConfigModel(db),
		UserReputationProfileModel: model.NewUserReputationProfileModel(db),
	}
}

func TestAdminGetUserListIncludesMemberStatusExpiryAndReputation(t *testing.T) {
	svcCtx := newAdminUserListTestSvc(t)
	activeExpiresAt := logicx.NowUTC8().Add(24 * time.Hour)
	expiredAt := logicx.NowUTC8().Add(-24 * time.Hour)
	banUntil := logicx.NowUTC8().Add(48 * time.Hour)
	reputationCfg := model.DefaultReputationConfig()
	reputationCfg.SetBaseRules(model.ReputationBaseRules{
		MaxScore:         120,
		InitialScore:     88,
		BanThreshold:     60,
		BanDurationHours: 24,
		MinScore:         0,
	})
	if err := svcCtx.ReputationConfigModel.Upsert(reputationCfg); err != nil {
		t.Fatalf("seed reputation config: %v", err)
	}
	users := []model.User{
		{Id: 1001, Nickname: "会员用户", Status: 1, MemberExpiresAt: &activeExpiresAt},
		{Id: 1002, Nickname: "过期用户", Status: 1, MemberExpiresAt: &expiredAt},
		{Id: 1003, Nickname: "普通用户", Status: 1},
	}
	for _, user := range users {
		u := user
		if err := svcCtx.UserModel.Create(&u); err != nil {
			t.Fatalf("create user %d: %v", u.Id, err)
		}
	}
	if err := svcCtx.UserReputationProfileModel.Save(&model.UserReputationProfile{
		UserID:          1001,
		ReputationScore: 73,
		BanUntil:        &banUntil,
	}); err != nil {
		t.Fatalf("seed reputation profile: %v", err)
	}

	resp, err := NewAdminGetUserListLogic(context.Background(), svcCtx).AdminGetUserList(&types.AdminUserListReq{
		Page:     1,
		PageSize: 20,
		Status:   -1,
	})
	if err != nil {
		t.Fatalf("get user list: %v", err)
	}
	if !resp.Success || len(resp.List) != 3 {
		t.Fatalf("unexpected user list response: %#v", resp)
	}

	memberStatuses := map[int64]string{}
	memberExpiry := map[int64]string{}
	reputationScores := map[int64]int{}
	banUntilValues := map[int64]string{}
	for _, item := range resp.List {
		memberStatuses[item.Id] = item.MemberStatus
		memberExpiry[item.Id] = item.MemberExpiresAt
		reputationScores[item.Id] = item.ReputationScore
		banUntilValues[item.Id] = item.BanUntil
	}

	if memberStatuses[1001] != "会员中" {
		t.Fatalf("expected active member status, got %#v", resp.List)
	}
	if memberExpiry[1001] == "" {
		t.Fatalf("expected active member expiry to be populated, got %#v", resp.List)
	}
	if memberStatuses[1002] != "已到期" {
		t.Fatalf("expected expired member status, got %#v", resp.List)
	}
	if memberStatuses[1003] != "未开通" {
		t.Fatalf("expected normal user status, got %#v", resp.List)
	}
	if reputationScores[1001] != 73 {
		t.Fatalf("expected profiled reputation score, got %#v", resp.List)
	}
	if banUntilValues[1001] == "" {
		t.Fatalf("expected profiled ban_until, got %#v", resp.List)
	}
	if reputationScores[1002] != 88 || reputationScores[1003] != 88 {
		t.Fatalf("expected initial score fallback for users without profile, got %#v", resp.List)
	}
	if banUntilValues[1002] != "" || banUntilValues[1003] != "" {
		t.Fatalf("expected empty ban_until for users without profile, got %#v", resp.List)
	}
}

func TestAdminGetUserListUsesRecoveredReputationScore(t *testing.T) {
	svcCtx := newAdminUserListTestSvc(t)
	now := time.Date(2026, 4, 13, 21, 0, 0, 0, logicx.UTC8Location)
	lastRecoveredAt := now.Add(-(2*time.Hour + 5*time.Minute))
	originalNow := adminUserListNow
	adminUserListNow = func() time.Time { return now }
	defer func() {
		adminUserListNow = originalNow
	}()

	reputationCfg := model.DefaultReputationConfig()
	reputationCfg.SetBaseRules(model.ReputationBaseRules{
		MaxScore:         120,
		InitialScore:     88,
		BanThreshold:     60,
		BanDurationHours: 24,
		MinScore:         0,
	})
	reputationCfg.SetRecoveryRules(model.ReputationRecoveryRules{
		Enabled:         true,
		RecoverPerHour:  3,
		RecoverMaxScore: 120,
	})
	if err := svcCtx.ReputationConfigModel.Upsert(reputationCfg); err != nil {
		t.Fatalf("seed reputation config: %v", err)
	}

	user := model.User{Id: 2001, Nickname: "恢复用户", Status: 1}
	if err := svcCtx.UserModel.Create(&user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := svcCtx.UserReputationProfileModel.Save(&model.UserReputationProfile{
		UserID:          2001,
		ReputationScore: 70,
		LastRecoveredAt: &lastRecoveredAt,
	}); err != nil {
		t.Fatalf("seed reputation profile: %v", err)
	}

	resp, err := NewAdminGetUserListLogic(context.Background(), svcCtx).AdminGetUserList(&types.AdminUserListReq{
		Page:     1,
		PageSize: 20,
		Status:   -1,
	})
	if err != nil {
		t.Fatalf("get user list: %v", err)
	}
	if !resp.Success || len(resp.List) != 1 {
		t.Fatalf("unexpected user list response: %#v", resp)
	}
	if resp.List[0].ReputationScore != 76 {
		t.Fatalf("expected recovered reputation score 76, got %#v", resp.List[0])
	}

	profile, err := svcCtx.UserReputationProfileModel.FindByUserID(2001)
	if err != nil {
		t.Fatalf("reload reputation profile: %v", err)
	}
	if profile == nil || profile.ReputationScore != 70 {
		t.Fatalf("expected db profile score to remain unchanged, got %+v", profile)
	}
}
