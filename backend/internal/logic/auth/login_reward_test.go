package auth

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/sms"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAuthRewardTestSvc(t *testing.T) (*svc.ServiceContext, *miniredis.Miniredis) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserOauth{}, &model.FavoriteVenueRewardConfig{}); err != nil {
		t.Fatalf("prepare auth reward schema: %v", err)
	}

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	cfg := config.Config{}
	cfg.Auth.AccessSecret = "test-secret"
	cfg.Auth.AccessExpire = 3600

	return &svc.ServiceContext{
		DB:                             db,
		Config:                         cfg,
		UserModel:                      model.NewUserModel(db),
		OauthModel:                     model.NewUserOauthModel(db),
		FavoriteVenueRewardConfigModel: model.NewFavoriteVenueRewardConfigModel(db),
		CodeManager:                    sms.NewCodeManager(rdb),
	}, mr
}

func assertWelcomeRewardDisabled(t *testing.T, user *model.User) {
	t.Helper()
	if user == nil {
		t.Fatalf("expected user, got nil")
	}
	if user.MemberExpiresAt != nil {
		t.Fatalf("expected no member expiry on new user, got %#v", user.MemberExpiresAt)
	}
}

func assertWelcomeRewardDuration(t *testing.T, user *model.User, rewardDays int) {
	t.Helper()
	if user == nil || user.MemberExpiresAt == nil {
		t.Fatalf("expected member expiry on new user, got %#v", user)
	}

	duration := user.MemberExpiresAt.Sub(user.CreatedAt)
	minExpected := time.Duration(rewardDays)*24*time.Hour - time.Minute
	maxExpected := time.Duration(rewardDays)*24*time.Hour + time.Minute
	if duration < minExpected || duration > maxExpected {
		t.Fatalf("expected welcome reward around %d days, got %s", rewardDays, duration)
	}
}

func TestLoginCreatesNewUserWithoutWelcomeRewardByDefault(t *testing.T) {
	svcCtx, _ := newAuthRewardTestSvc(t)
	ctx := context.Background()

	if err := svcCtx.CodeManager.SaveCode(ctx, "13800138000", "123456"); err != nil {
		t.Fatalf("save sms code: %v", err)
	}

	logic := NewLoginLogic(ctx, svcCtx)
	resp, err := logic.Login(&types.LoginReq{
		Phone:   "13800138000",
		SmsCode: "123456",
	})
	if err != nil {
		t.Fatalf("login new user: %v", err)
	}
	if !resp.Success || resp.UserInfo == nil {
		t.Fatalf("expected successful login response, got %#v", resp)
	}

	user, err := svcCtx.UserModel.FindByPhone("13800138000")
	if err != nil {
		t.Fatalf("find created user: %v", err)
	}
	assertWelcomeRewardDisabled(t, user)
}

func TestLoginCreatesNewUserWithConfiguredWelcomeReward(t *testing.T) {
	svcCtx, _ := newAuthRewardTestSvc(t)
	ctx := context.Background()

	if err := svcCtx.FavoriteVenueRewardConfigModel.Upsert(&model.FavoriteVenueRewardConfig{
		ActivityKey:       model.WelcomeMemberRewardActivityKey,
		Enabled:           true,
		RewardDays:        14,
		NewUserWindowDays: 7,
		PopupEnabled:      false,
	}); err != nil {
		t.Fatalf("seed explicit welcome reward config: %v", err)
	}

	if err := svcCtx.CodeManager.SaveCode(ctx, "13800138000", "123456"); err != nil {
		t.Fatalf("save sms code: %v", err)
	}

	logic := NewLoginLogic(ctx, svcCtx)
	resp, err := logic.Login(&types.LoginReq{
		Phone:   "13800138000",
		SmsCode: "123456",
	})
	if err != nil {
		t.Fatalf("login new user: %v", err)
	}
	if !resp.Success || resp.UserInfo == nil {
		t.Fatalf("expected successful login response, got %#v", resp)
	}

	user, err := svcCtx.UserModel.FindByPhone("13800138000")
	if err != nil {
		t.Fatalf("find created user: %v", err)
	}
	assertWelcomeRewardDuration(t, user, 14)
}

func TestLoginAvatarSemantics(t *testing.T) {
	svcCtx, _ := newAuthRewardTestSvc(t)
	ctx := context.Background()

	if err := svcCtx.CodeManager.SaveCode(ctx, "13800138000", "123456"); err != nil {
		t.Fatalf("save new user sms code: %v", err)
	}
	resp, err := NewLoginLogic(ctx, svcCtx).Login(&types.LoginReq{
		Phone:   "13800138000",
		SmsCode: "123456",
	})
	if err != nil {
		t.Fatalf("login new user: %v", err)
	}
	if resp.UserInfo == nil || resp.UserInfo.Avatar != "" {
		t.Fatalf("new user should keep avatar empty, got %#v", resp.UserInfo)
	}

	realAvatar := "https://img.example.com/real-avatar.png"
	phone := "13900139000"
	if err := svcCtx.UserModel.Create(&model.User{
		Phone:    &phone,
		Nickname: "已有头像用户",
		Avatar:   realAvatar,
		Status:   1,
	}); err != nil {
		t.Fatalf("create existing avatar user: %v", err)
	}
	if err := svcCtx.CodeManager.SaveCode(ctx, phone, "654321"); err != nil {
		t.Fatalf("save existing user sms code: %v", err)
	}
	resp, err = NewLoginLogic(ctx, svcCtx).Login(&types.LoginReq{
		Phone:   phone,
		SmsCode: "654321",
	})
	if err != nil {
		t.Fatalf("login existing user: %v", err)
	}
	if resp.UserInfo == nil || resp.UserInfo.Avatar != realAvatar {
		t.Fatalf("existing avatar should remain unchanged, got %#v", resp.UserInfo)
	}
}

func TestLoginByOauthCreatesNewUserWithoutWelcomeRewardByDefault(t *testing.T) {
	svcCtx, _ := newAuthRewardTestSvc(t)
	ctx := context.Background()

	logic := NewLoginByOauthLogic(ctx, svcCtx)
	resp, err := logic.LoginByOauth(&types.LoginByOauthReq{
		Provider: "huawei",
		OpenId:   "openid-1001",
		NickName: "华为新用户",
	})
	if err != nil {
		t.Fatalf("oauth login new user: %v", err)
	}
	if !resp.Success || resp.UserInfo == nil || !resp.NeedBindPhone {
		t.Fatalf("expected successful oauth login response, got %#v", resp)
	}

	user, err := svcCtx.UserModel.FindById(resp.UserInfo.Id)
	if err != nil {
		t.Fatalf("find oauth user: %v", err)
	}
	assertWelcomeRewardDisabled(t, user)

	oauth, err := svcCtx.OauthModel.FindByProviderAndOpenId("huawei", "openid-1001")
	if err != nil {
		t.Fatalf("find oauth record: %v", err)
	}
	if oauth == nil || oauth.UserId != resp.UserInfo.Id {
		t.Fatalf("expected oauth record bound to new user, got %#v", oauth)
	}
}

func TestLoginByOauthCreatesNewUserWithConfiguredWelcomeReward(t *testing.T) {
	svcCtx, _ := newAuthRewardTestSvc(t)
	ctx := context.Background()

	if err := svcCtx.FavoriteVenueRewardConfigModel.Upsert(&model.FavoriteVenueRewardConfig{
		ActivityKey:       model.WelcomeMemberRewardActivityKey,
		Enabled:           true,
		RewardDays:        14,
		NewUserWindowDays: 7,
		PopupEnabled:      false,
	}); err != nil {
		t.Fatalf("seed explicit welcome reward config: %v", err)
	}

	logic := NewLoginByOauthLogic(ctx, svcCtx)
	resp, err := logic.LoginByOauth(&types.LoginByOauthReq{
		Provider: "huawei",
		OpenId:   "openid-1001",
		NickName: "华为新用户",
	})
	if err != nil {
		t.Fatalf("oauth login new user with explicit welcome reward: %v", err)
	}
	if !resp.Success || resp.UserInfo == nil || !resp.NeedBindPhone {
		t.Fatalf("expected successful oauth login response, got %#v", resp)
	}

	user, err := svcCtx.UserModel.FindById(resp.UserInfo.Id)
	if err != nil {
		t.Fatalf("find oauth user: %v", err)
	}
	assertWelcomeRewardDuration(t, user, 14)
}
