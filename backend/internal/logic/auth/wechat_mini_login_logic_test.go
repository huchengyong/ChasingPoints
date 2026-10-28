package auth

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/pkg/wechatmini"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeWechatMiniClient struct {
	identity   *wechatmini.Identity
	phone      string
	err        error
	phoneErr   error
	phoneCalls int
}

func (f *fakeWechatMiniClient) ExchangeLoginCode(context.Context, string) (*wechatmini.Identity, error) {
	return f.identity, f.err
}

func (f *fakeWechatMiniClient) GetPhoneNumber(context.Context, string) (string, error) {
	f.phoneCalls++
	if f.phoneErr != nil {
		return "", f.phoneErr
	}
	return f.phone, f.err
}

func newWechatMiniAuthTestSvc(t *testing.T, client wechatmini.Client) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			phone TEXT UNIQUE,
			nickname TEXT NOT NULL DEFAULT '',
			avatar TEXT NOT NULL DEFAULT '',
			status INTEGER NOT NULL DEFAULT 1,
			push_token TEXT NOT NULL DEFAULT '',
			member_expires_at DATETIME,
			hide_match_record BOOLEAN NOT NULL DEFAULT false,
			friends_only_challenges BOOLEAN NOT NULL DEFAULT false,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME
		);
		CREATE TABLE user_oauth (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			provider TEXT NOT NULL,
			open_id TEXT NOT NULL,
			union_id TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(provider, open_id)
		);
	`).Error; err != nil {
		t.Fatalf("prepare auth schema: %v", err)
	}
	if err := db.AutoMigrate(&model.FavoriteVenueRewardConfig{}); err != nil {
		t.Fatalf("prepare welcome reward schema: %v", err)
	}

	cfg := config.Config{}
	cfg.Auth.AccessSecret = "wechat-mini-test-secret"
	cfg.Auth.AccessExpire = 3600
	cfg.Auth.RefreshExpire = 7200

	return &svc.ServiceContext{
		DB:                             db,
		Config:                         cfg,
		UserModel:                      model.NewUserModel(db),
		OauthModel:                     model.NewUserOauthModel(db),
		FavoriteVenueRewardConfigModel: model.NewFavoriteVenueRewardConfigModel(db),
		WechatMiniClient:               client,
	}
}

func TestWechatMiniLoginCreatesPhoneFreeUserAndOAuthAssociation(t *testing.T) {
	client := &fakeWechatMiniClient{identity: &wechatmini.Identity{
		OpenID:  "wechat-openid-1",
		UnionID: "wechat-unionid-1",
	}}
	svcCtx := newWechatMiniAuthTestSvc(t, client)

	resp, err := NewWechatMiniLoginLogic(context.Background(), svcCtx).WechatMiniLogin(&types.WechatMiniLoginReq{
		Code: "login-code",
	})
	if err != nil {
		t.Fatalf("wechat mini login: %v", err)
	}
	if !resp.Success || !resp.NeedBindPhone || resp.UserInfo == nil || !defaultNicknamePattern.MatchString(resp.UserInfo.Nickname) {
		t.Fatalf("unexpected login response: %#v", resp)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatalf("expected token pair, got %#v", resp)
	}

	oauth, err := svcCtx.OauthModel.FindByProviderAndOpenId("weixin_mini_program", "wechat-openid-1")
	if err != nil {
		t.Fatalf("find oauth association: %v", err)
	}
	if oauth == nil || oauth.UserId != resp.UserInfo.Id {
		t.Fatalf("unexpected oauth association: %#v", oauth)
	}

	user, err := svcCtx.UserModel.FindById(resp.UserInfo.Id)
	if err != nil {
		t.Fatalf("find created user: %v", err)
	}
	if user == nil || user.MemberExpiresAt != nil {
		t.Fatalf("expected no member expiry by default, got %#v", user)
	}
}

func TestWechatMiniLoginCreatesNewUserWithConfiguredWelcomeReward(t *testing.T) {
	client := &fakeWechatMiniClient{identity: &wechatmini.Identity{OpenID: "wechat-openid-configured", UnionID: "wechat-unionid-configured"}}
	svcCtx := newWechatMiniAuthTestSvc(t, client)

	if err := svcCtx.FavoriteVenueRewardConfigModel.Upsert(&model.FavoriteVenueRewardConfig{
		ActivityKey:       model.WelcomeMemberRewardActivityKey,
		Enabled:           true,
		RewardDays:        14,
		NewUserWindowDays: 7,
		PopupEnabled:      false,
	}); err != nil {
		t.Fatalf("seed explicit welcome reward config: %v", err)
	}

	resp, err := NewWechatMiniLoginLogic(context.Background(), svcCtx).WechatMiniLogin(&types.WechatMiniLoginReq{
		Code: "login-code",
	})
	if err != nil {
		t.Fatalf("wechat mini login with explicit welcome reward: %v", err)
	}
	if !resp.Success || !resp.NeedBindPhone || resp.UserInfo == nil || !defaultNicknamePattern.MatchString(resp.UserInfo.Nickname) {
		t.Fatalf("unexpected login response: %#v", resp)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatalf("expected token pair, got %#v", resp)
	}

	user, err := svcCtx.UserModel.FindById(resp.UserInfo.Id)
	if err != nil {
		t.Fatalf("find created user: %v", err)
	}
	if user == nil || user.MemberExpiresAt == nil {
		t.Fatalf("expected welcome reward applied with config, got %#v", user)
	}

	duration := user.MemberExpiresAt.Sub(user.CreatedAt)
	const expectedDays = 14
	minExpected := expectedDays*24*time.Hour - time.Minute
	maxExpected := expectedDays*24*time.Hour + time.Minute
	if duration < minExpected || duration > maxExpected {
		t.Fatalf("expected welcome reward around %d days, got %s", expectedDays, duration)
	}
}

func TestWechatMiniLoginReusesExistingOAuthAssociation(t *testing.T) {
	client := &fakeWechatMiniClient{identity: &wechatmini.Identity{OpenID: "wechat-openid-existing"}}
	svcCtx := newWechatMiniAuthTestSvc(t, client)
	existingUser := &model.User{Nickname: "已关联用户", Status: 1}
	if err := svcCtx.UserModel.Create(existingUser); err != nil {
		t.Fatalf("create existing user: %v", err)
	}
	if err := svcCtx.OauthModel.Create(&model.UserOauth{
		UserId:   existingUser.Id,
		Provider: wechatMiniProvider,
		OpenId:   "wechat-openid-existing",
	}); err != nil {
		t.Fatalf("create existing oauth: %v", err)
	}

	resp, err := NewWechatMiniLoginLogic(context.Background(), svcCtx).WechatMiniLogin(&types.WechatMiniLoginReq{Code: "login-code"})
	if err != nil {
		t.Fatalf("wechat mini login: %v", err)
	}
	if !resp.Success || resp.UserInfo == nil || resp.UserInfo.Id != existingUser.Id || resp.UserInfo.Nickname != existingUser.Nickname {
		t.Fatalf("expected existing user login, got %#v", resp)
	}

	var count int64
	if err := svcCtx.DB.Model(&model.User{}).Count(&count).Error; err != nil {
		t.Fatalf("count users: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected no additional user, got %d users", count)
	}
}

func TestWechatMiniLoginRejectsDisabledExistingUserWithoutTokens(t *testing.T) {
	client := &fakeWechatMiniClient{identity: &wechatmini.Identity{OpenID: "wechat-openid-disabled"}}
	svcCtx := newWechatMiniAuthTestSvc(t, client)
	user := &model.User{Nickname: "停用微信用户", Status: 1}
	if err := svcCtx.UserModel.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := svcCtx.OauthModel.Create(&model.UserOauth{
		UserId:   user.Id,
		Provider: wechatMiniProvider,
		OpenId:   "wechat-openid-disabled",
	}); err != nil {
		t.Fatalf("create oauth: %v", err)
	}
	if err := svcCtx.DB.Model(&model.User{}).Where("id = ?", user.Id).Update("status", 0).Error; err != nil {
		t.Fatalf("disable user: %v", err)
	}

	resp, err := NewWechatMiniLoginLogic(context.Background(), svcCtx).WechatMiniLogin(&types.WechatMiniLoginReq{Code: "login-code"})
	if err != nil {
		t.Fatalf("disabled user login should return a safe response: %v", err)
	}
	if resp == nil || resp.Success || resp.AccessToken != "" || resp.RefreshToken != "" || resp.Message == "" {
		t.Fatalf("disabled WeChat user received a session: %#v", resp)
	}
}

func TestWechatMiniLoginFailureDoesNotCreateUser(t *testing.T) {
	client := &fakeWechatMiniClient{err: &wechatmini.Error{Kind: wechatmini.ErrorKindRejected}}
	svcCtx := newWechatMiniAuthTestSvc(t, client)

	resp, err := NewWechatMiniLoginLogic(context.Background(), svcCtx).WechatMiniLogin(&types.WechatMiniLoginReq{Code: "expired-code"})
	if err != nil {
		t.Fatalf("wechat mini login: %v", err)
	}
	if resp.Success || resp.Message == "" {
		t.Fatalf("expected safe login failure, got %#v", resp)
	}

	var count int64
	if err := svcCtx.DB.Model(&model.User{}).Count(&count).Error; err != nil {
		t.Fatalf("count users: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no user after failed identity exchange, got %d", count)
	}
}

func countWechatMiniTestUsers(t *testing.T, svcCtx *svc.ServiceContext) int64 {
	t.Helper()
	var count int64
	if err := svcCtx.DB.Model(&model.User{}).Count(&count).Error; err != nil {
		t.Fatalf("count users: %v", err)
	}
	return count
}

func TestWechatMiniLoginWithPhoneCodeCreatesUserWithPhone(t *testing.T) {
	client := &fakeWechatMiniClient{
		identity: &wechatmini.Identity{OpenID: "wechat-openid-phone-new", UnionID: "wechat-unionid-phone-new"},
		phone:    "13800138001",
	}
	svcCtx := newWechatMiniAuthTestSvc(t, client)

	resp, err := NewWechatMiniLoginLogic(context.Background(), svcCtx).WechatMiniLogin(&types.WechatMiniLoginReq{
		Code:      "login-code",
		PhoneCode: "phone-code",
	})
	if err != nil {
		t.Fatalf("wechat mini login: %v", err)
	}
	if !resp.Success || resp.NeedBindPhone || resp.UserInfo == nil || resp.UserInfo.Phone != "138****8001" {
		t.Fatalf("expected a phone-bound login, got %#v", resp)
	}
	if !defaultNicknamePattern.MatchString(resp.UserInfo.Nickname) || resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatalf("unexpected login response: %#v", resp)
	}

	user, err := svcCtx.UserModel.FindByPhone("13800138001")
	if err != nil || user == nil || user.Id != resp.UserInfo.Id {
		t.Fatalf("expected the created user to own the phone, got %#v, err %v", user, err)
	}
	oauth, err := svcCtx.OauthModel.FindByProviderAndOpenId(wechatMiniProvider, "wechat-openid-phone-new")
	if err != nil || oauth == nil || oauth.UserId != user.Id {
		t.Fatalf("expected oauth linked to the created user, got %#v, err %v", oauth, err)
	}
}

func TestWechatMiniLoginWithPhoneCodeLinksExistingPhoneAccount(t *testing.T) {
	phone := "13800138002"
	client := &fakeWechatMiniClient{identity: &wechatmini.Identity{OpenID: "wechat-openid-phone-link"}, phone: phone}
	svcCtx := newWechatMiniAuthTestSvc(t, client)
	phoneUser := &model.User{Phone: &phone, Nickname: "手机号用户", Status: 1}
	if err := svcCtx.UserModel.Create(phoneUser); err != nil {
		t.Fatalf("create phone user: %v", err)
	}

	resp, err := NewWechatMiniLoginLogic(context.Background(), svcCtx).WechatMiniLogin(&types.WechatMiniLoginReq{
		Code:      "login-code",
		PhoneCode: "phone-code",
	})
	if err != nil {
		t.Fatalf("wechat mini login: %v", err)
	}
	if !resp.Success || resp.NeedBindPhone || resp.UserInfo == nil || resp.UserInfo.Id != phoneUser.Id || resp.UserInfo.Nickname != "手机号用户" {
		t.Fatalf("expected login into the existing phone account, got %#v", resp)
	}
	if count := countWechatMiniTestUsers(t, svcCtx); count != 1 {
		t.Fatalf("expected no additional user, got %d users", count)
	}
	oauth, err := svcCtx.OauthModel.FindByProviderAndOpenId(wechatMiniProvider, "wechat-openid-phone-link")
	if err != nil || oauth == nil || oauth.UserId != phoneUser.Id {
		t.Fatalf("expected oauth linked to the phone account, got %#v, err %v", oauth, err)
	}
}

func TestWechatMiniLoginWithPhoneCodeRejectsDisabledPhoneAccount(t *testing.T) {
	phone := "13800138003"
	client := &fakeWechatMiniClient{identity: &wechatmini.Identity{OpenID: "wechat-openid-phone-disabled"}, phone: phone}
	svcCtx := newWechatMiniAuthTestSvc(t, client)
	phoneUser := &model.User{Phone: &phone, Nickname: "停用手机号用户", Status: 1}
	if err := svcCtx.UserModel.Create(phoneUser); err != nil {
		t.Fatalf("create phone user: %v", err)
	}
	if err := svcCtx.DB.Model(&model.User{}).Where("id = ?", phoneUser.Id).Update("status", 0).Error; err != nil {
		t.Fatalf("disable phone user: %v", err)
	}

	resp, err := NewWechatMiniLoginLogic(context.Background(), svcCtx).WechatMiniLogin(&types.WechatMiniLoginReq{
		Code:      "login-code",
		PhoneCode: "phone-code",
	})
	if err != nil {
		t.Fatalf("disabled phone account login should return a safe response: %v", err)
	}
	if resp == nil || resp.Success || resp.AccessToken != "" || resp.RefreshToken != "" || resp.Message != loginUnavailableMessage {
		t.Fatalf("disabled phone account received a session: %#v", resp)
	}
	oauth, err := svcCtx.OauthModel.FindByProviderAndOpenId(wechatMiniProvider, "wechat-openid-phone-disabled")
	if err != nil || oauth != nil {
		t.Fatalf("oauth must not be linked to a disabled account, got %#v, err %v", oauth, err)
	}
	if count := countWechatMiniTestUsers(t, svcCtx); count != 1 {
		t.Fatalf("expected no additional user, got %d users", count)
	}
}

func TestWechatMiniLoginPhoneAuthorizationFailureDoesNotCreateUser(t *testing.T) {
	client := &fakeWechatMiniClient{
		identity: &wechatmini.Identity{OpenID: "wechat-openid-phone-failed"},
		phoneErr: &wechatmini.Error{Kind: wechatmini.ErrorKindRejected},
	}
	svcCtx := newWechatMiniAuthTestSvc(t, client)

	resp, err := NewWechatMiniLoginLogic(context.Background(), svcCtx).WechatMiniLogin(&types.WechatMiniLoginReq{
		Code:      "login-code",
		PhoneCode: "expired-phone-code",
	})
	if err != nil {
		t.Fatalf("wechat mini login: %v", err)
	}
	if resp.Success || resp.Message == "" || resp.AccessToken != "" {
		t.Fatalf("expected safe phone authorization failure, got %#v", resp)
	}
	if count := countWechatMiniTestUsers(t, svcCtx); count != 0 {
		t.Fatalf("expected no user after phone authorization failure, got %d", count)
	}
	oauth, err := svcCtx.OauthModel.FindByProviderAndOpenId(wechatMiniProvider, "wechat-openid-phone-failed")
	if err != nil || oauth != nil {
		t.Fatalf("oauth must not be created after phone authorization failure, got %#v, err %v", oauth, err)
	}
}

func TestWechatMiniLoginRejectsMalformedAuthorizedPhone(t *testing.T) {
	client := &fakeWechatMiniClient{identity: &wechatmini.Identity{OpenID: "wechat-openid-phone-malformed"}, phone: "+8613800138004"}
	svcCtx := newWechatMiniAuthTestSvc(t, client)

	resp, err := NewWechatMiniLoginLogic(context.Background(), svcCtx).WechatMiniLogin(&types.WechatMiniLoginReq{
		Code:      "login-code",
		PhoneCode: "phone-code",
	})
	if err != nil {
		t.Fatalf("wechat mini login: %v", err)
	}
	if resp.Success || resp.Message != "手机号格式不正确" {
		t.Fatalf("expected malformed phone rejection, got %#v", resp)
	}
	if count := countWechatMiniTestUsers(t, svcCtx); count != 0 {
		t.Fatalf("expected no user, got %d", count)
	}
}

func TestWechatMiniLoginWithPhoneCodeBindsExistingPhoneFreeOAuthUser(t *testing.T) {
	client := &fakeWechatMiniClient{identity: &wechatmini.Identity{OpenID: "wechat-openid-phone-bind"}, phone: "13800138005"}
	svcCtx := newWechatMiniAuthTestSvc(t, client)
	phoneFreeUser := &model.User{Nickname: "旧微信用户", Status: 1}
	if err := svcCtx.UserModel.Create(phoneFreeUser); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := svcCtx.OauthModel.Create(&model.UserOauth{UserId: phoneFreeUser.Id, Provider: wechatMiniProvider, OpenId: "wechat-openid-phone-bind"}); err != nil {
		t.Fatalf("create oauth: %v", err)
	}

	resp, err := NewWechatMiniLoginLogic(context.Background(), svcCtx).WechatMiniLogin(&types.WechatMiniLoginReq{
		Code:      "login-code",
		PhoneCode: "phone-code",
	})
	if err != nil {
		t.Fatalf("wechat mini login: %v", err)
	}
	if !resp.Success || resp.NeedBindPhone || resp.UserInfo == nil || resp.UserInfo.Id != phoneFreeUser.Id || resp.UserInfo.Phone != "138****8005" {
		t.Fatalf("expected the existing user to receive the phone, got %#v", resp)
	}
}

func TestWechatMiniLoginWithPhoneCodeMergesPhoneFreeOAuthUserIntoPhoneAccount(t *testing.T) {
	phone := "13800138006"
	client := &fakeWechatMiniClient{identity: &wechatmini.Identity{OpenID: "wechat-openid-phone-merge"}, phone: phone}
	svcCtx := newWechatMiniAuthTestSvc(t, client)
	phoneFreeUser := &model.User{Nickname: "旧微信用户", Status: 1}
	if err := svcCtx.UserModel.Create(phoneFreeUser); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := svcCtx.OauthModel.Create(&model.UserOauth{UserId: phoneFreeUser.Id, Provider: wechatMiniProvider, OpenId: "wechat-openid-phone-merge"}); err != nil {
		t.Fatalf("create oauth: %v", err)
	}
	phoneUser := &model.User{Phone: &phone, Nickname: "手机号用户", Status: 1}
	if err := svcCtx.UserModel.Create(phoneUser); err != nil {
		t.Fatalf("create phone user: %v", err)
	}

	resp, err := NewWechatMiniLoginLogic(context.Background(), svcCtx).WechatMiniLogin(&types.WechatMiniLoginReq{
		Code:      "login-code",
		PhoneCode: "phone-code",
	})
	if err != nil {
		t.Fatalf("wechat mini login: %v", err)
	}
	if !resp.Success || resp.NeedBindPhone || resp.UserInfo == nil || resp.UserInfo.Id != phoneUser.Id {
		t.Fatalf("expected a session for the merged phone account, got %#v", resp)
	}
	oauth, err := svcCtx.OauthModel.FindByProviderAndOpenId(wechatMiniProvider, "wechat-openid-phone-merge")
	if err != nil || oauth == nil || oauth.UserId != phoneUser.Id {
		t.Fatalf("expected oauth moved to the phone account, got %#v, err %v", oauth, err)
	}
	if removed, err := svcCtx.UserModel.FindById(phoneFreeUser.Id); err != nil || removed != nil {
		t.Fatalf("expected the phone-free user removed, got %#v, err %v", removed, err)
	}
}

func TestWechatMiniLoginSkipsPhoneExchangeForUsersWithPhone(t *testing.T) {
	phone := "13800138007"
	client := &fakeWechatMiniClient{identity: &wechatmini.Identity{OpenID: "wechat-openid-phone-existing"}, phone: "13900139007"}
	svcCtx := newWechatMiniAuthTestSvc(t, client)
	user := &model.User{Phone: &phone, Nickname: "已绑定用户", Status: 1}
	if err := svcCtx.UserModel.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := svcCtx.OauthModel.Create(&model.UserOauth{UserId: user.Id, Provider: wechatMiniProvider, OpenId: "wechat-openid-phone-existing"}); err != nil {
		t.Fatalf("create oauth: %v", err)
	}

	resp, err := NewWechatMiniLoginLogic(context.Background(), svcCtx).WechatMiniLogin(&types.WechatMiniLoginReq{
		Code:      "login-code",
		PhoneCode: "phone-code",
	})
	if err != nil {
		t.Fatalf("wechat mini login: %v", err)
	}
	if !resp.Success || resp.UserInfo == nil || resp.UserInfo.Id != user.Id || resp.UserInfo.Phone != "138****8007" {
		t.Fatalf("expected login to keep the bound phone, got %#v", resp)
	}
	if client.phoneCalls != 0 {
		t.Fatalf("expected no phone exchange for a user that already has a phone, got %d calls", client.phoneCalls)
	}
}

func TestWechatMiniBindPhoneMergesIntoPhoneAccountAndReturnsNewSession(t *testing.T) {
	phone := "13800138000"
	client := &fakeWechatMiniClient{phone: phone}
	svcCtx := newWechatMiniAuthTestSvc(t, client)

	temporaryUser := &model.User{Nickname: "微信用户", Status: 1}
	if err := svcCtx.UserModel.Create(temporaryUser); err != nil {
		t.Fatalf("create temporary user: %v", err)
	}
	if err := svcCtx.OauthModel.Create(&model.UserOauth{
		UserId:   temporaryUser.Id,
		Provider: wechatMiniProvider,
		OpenId:   "wechat-openid-merge",
	}); err != nil {
		t.Fatalf("create temporary oauth: %v", err)
	}

	phoneUser := &model.User{Phone: &phone, Nickname: "手机号用户", Status: 1}
	if err := svcCtx.UserModel.Create(phoneUser); err != nil {
		t.Fatalf("create phone user: %v", err)
	}

	ctx := context.WithValue(context.Background(), "user_id", float64(temporaryUser.Id))
	resp, err := NewWechatMiniBindPhoneLogic(ctx, svcCtx).WechatMiniBindPhone(&types.WechatMiniBindPhoneReq{Code: "phone-code"})
	if err != nil {
		t.Fatalf("wechat mini bind phone: %v", err)
	}
	if !resp.Success || !resp.MergedAccount || resp.NeedBindPhone || resp.UserInfo == nil || resp.UserInfo.Id != phoneUser.Id {
		t.Fatalf("unexpected bind response: %#v", resp)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatalf("expected replacement session, got %#v", resp)
	}

	oauth, err := svcCtx.OauthModel.FindByProviderAndOpenId(wechatMiniProvider, "wechat-openid-merge")
	if err != nil {
		t.Fatalf("find migrated oauth: %v", err)
	}
	if oauth == nil || oauth.UserId != phoneUser.Id {
		t.Fatalf("expected oauth moved to phone user, got %#v", oauth)
	}

	deletedUser, err := svcCtx.UserModel.FindById(temporaryUser.Id)
	if err != nil {
		t.Fatalf("find deleted temporary user: %v", err)
	}
	if deletedUser != nil {
		t.Fatalf("expected temporary user deleted, got %#v", deletedUser)
	}
}

func TestWechatMiniBindPhoneRejectsDisabledMergeTargetWithoutChangingSource(t *testing.T) {
	phone := "13500135000"
	svcCtx := newWechatMiniAuthTestSvc(t, &fakeWechatMiniClient{phone: phone})

	temporaryUser := &model.User{Nickname: "微信用户", Status: 1}
	if err := svcCtx.UserModel.Create(temporaryUser); err != nil {
		t.Fatalf("create temporary user: %v", err)
	}
	if err := svcCtx.OauthModel.Create(&model.UserOauth{
		UserId:   temporaryUser.Id,
		Provider: wechatMiniProvider,
		OpenId:   "wechat-openid-disabled-target",
	}); err != nil {
		t.Fatalf("create temporary oauth: %v", err)
	}

	phoneUser := &model.User{Phone: &phone, Nickname: "停用手机号用户", Status: 1}
	if err := svcCtx.UserModel.Create(phoneUser); err != nil {
		t.Fatalf("create phone user: %v", err)
	}
	if err := svcCtx.DB.Model(&model.User{}).Where("id = ?", phoneUser.Id).Update("status", 0).Error; err != nil {
		t.Fatalf("disable phone user: %v", err)
	}

	ctx := context.WithValue(context.Background(), "user_id", float64(temporaryUser.Id))
	resp, err := NewWechatMiniBindPhoneLogic(ctx, svcCtx).WechatMiniBindPhone(&types.WechatMiniBindPhoneReq{Code: "phone-code"})
	if err != nil {
		t.Fatalf("wechat mini bind disabled phone user: %v", err)
	}
	if resp.Success || resp.MergedAccount || resp.AccessToken != "" || resp.RefreshToken != "" {
		t.Fatalf("expected disabled target merge rejection, got %#v", resp)
	}

	oauth, err := svcCtx.OauthModel.FindByProviderAndOpenId(wechatMiniProvider, "wechat-openid-disabled-target")
	if err != nil {
		t.Fatalf("find unchanged oauth: %v", err)
	}
	if oauth == nil || oauth.UserId != temporaryUser.Id {
		t.Fatalf("oauth ownership changed for disabled target: %#v", oauth)
	}
	retainedUser, err := svcCtx.UserModel.FindById(temporaryUser.Id)
	if err != nil {
		t.Fatalf("find retained temporary user: %v", err)
	}
	if retainedUser == nil {
		t.Fatal("temporary user was deleted for a disabled merge target")
	}
}

func TestWechatMiniBindPhoneUpdatesCurrentUserWithoutReplacingSession(t *testing.T) {
	phone := "13900139000"
	svcCtx := newWechatMiniAuthTestSvc(t, &fakeWechatMiniClient{phone: phone})
	user := &model.User{Nickname: "微信用户", Status: 1}
	if err := svcCtx.UserModel.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	ctx := context.WithValue(context.Background(), "user_id", float64(user.Id))
	resp, err := NewWechatMiniBindPhoneLogic(ctx, svcCtx).WechatMiniBindPhone(&types.WechatMiniBindPhoneReq{Code: "phone-code"})
	if err != nil {
		t.Fatalf("wechat mini bind phone: %v", err)
	}
	if !resp.Success || resp.MergedAccount || resp.NeedBindPhone || resp.UserInfo == nil {
		t.Fatalf("unexpected bind response: %#v", resp)
	}
	if resp.AccessToken != "" || resp.RefreshToken != "" {
		t.Fatalf("normal binding must not replace session: %#v", resp)
	}
	if resp.UserInfo.Phone != "139****9000" {
		t.Fatalf("expected masked phone, got %s", resp.UserInfo.Phone)
	}
}

func TestWechatMiniBindPhoneAllowsRepeatedBindingForCurrentUser(t *testing.T) {
	phone := "13700137000"
	svcCtx := newWechatMiniAuthTestSvc(t, &fakeWechatMiniClient{phone: phone})
	user := &model.User{Nickname: "微信用户", Status: 1}
	if err := svcCtx.UserModel.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	ctx := context.WithValue(context.Background(), "user_id", float64(user.Id))
	logic := NewWechatMiniBindPhoneLogic(ctx, svcCtx)
	for attempt := 0; attempt < 2; attempt++ {
		resp, err := logic.WechatMiniBindPhone(&types.WechatMiniBindPhoneReq{Code: "phone-code"})
		if err != nil {
			t.Fatalf("repeat bind attempt %d: %v", attempt+1, err)
		}
		if !resp.Success || resp.MergedAccount || resp.NeedBindPhone {
			t.Fatalf("unexpected repeat bind response: %#v", resp)
		}
	}

	boundUser, err := svcCtx.UserModel.FindById(user.Id)
	if err != nil {
		t.Fatalf("find bound user: %v", err)
	}
	if boundUser == nil || boundUser.Phone == nil || *boundUser.Phone != phone {
		t.Fatalf("expected the same user to remain bound, got %#v", boundUser)
	}
}

func TestWechatMiniBindPhoneRollsBackMergeOnDeleteFailure(t *testing.T) {
	phone := "13600136000"
	svcCtx := newWechatMiniAuthTestSvc(t, &fakeWechatMiniClient{phone: phone})
	temporaryUser := &model.User{Nickname: "微信用户", Status: 1}
	if err := svcCtx.UserModel.Create(temporaryUser); err != nil {
		t.Fatalf("create temporary user: %v", err)
	}
	if err := svcCtx.OauthModel.Create(&model.UserOauth{
		UserId:   temporaryUser.Id,
		Provider: wechatMiniProvider,
		OpenId:   "wechat-openid-rollback",
	}); err != nil {
		t.Fatalf("create temporary oauth: %v", err)
	}

	phoneUser := &model.User{Phone: &phone, Nickname: "手机号用户", Status: 1}
	if err := svcCtx.UserModel.Create(phoneUser); err != nil {
		t.Fatalf("create phone user: %v", err)
	}
	if err := svcCtx.DB.Exec(`
		CREATE TRIGGER reject_user_soft_delete
		BEFORE UPDATE OF deleted_at ON users
		WHEN NEW.deleted_at IS NOT NULL
		BEGIN
			SELECT RAISE(ABORT, 'soft delete blocked');
		END;
	`).Error; err != nil {
		t.Fatalf("create soft delete trigger: %v", err)
	}

	ctx := context.WithValue(context.Background(), "user_id", float64(temporaryUser.Id))
	resp, err := NewWechatMiniBindPhoneLogic(ctx, svcCtx).WechatMiniBindPhone(&types.WechatMiniBindPhoneReq{Code: "phone-code"})
	if err != nil {
		t.Fatalf("bind phone with forced delete failure: %v", err)
	}
	if resp.Success || resp.MergedAccount {
		t.Fatalf("expected merge failure without a partial write, got %#v", resp)
	}

	oauth, err := svcCtx.OauthModel.FindByProviderAndOpenId(wechatMiniProvider, "wechat-openid-rollback")
	if err != nil {
		t.Fatalf("find unchanged oauth: %v", err)
	}
	if oauth == nil || oauth.UserId != temporaryUser.Id {
		t.Fatalf("oauth migration was not rolled back: %#v", oauth)
	}

	stillTemporaryUser, err := svcCtx.UserModel.FindById(temporaryUser.Id)
	if err != nil {
		t.Fatalf("find retained temporary user: %v", err)
	}
	if stillTemporaryUser == nil {
		t.Fatal("temporary user was deleted despite a failed merge")
	}
}

func TestLoginByOauthRejectsWechatMiniProvider(t *testing.T) {
	svcCtx := newWechatMiniAuthTestSvc(t, &fakeWechatMiniClient{})

	resp, err := NewLoginByOauthLogic(context.Background(), svcCtx).LoginByOauth(&types.LoginByOauthReq{
		Provider:   wechatMiniProvider,
		Credential: "forged-credential",
	})
	if err != nil {
		t.Fatalf("reject mini provider: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected mini provider to be rejected, got %#v", resp)
	}

	oauth, err := svcCtx.OauthModel.FindByProviderAndOpenId(wechatMiniProvider, "forged-openid")
	if err != nil {
		t.Fatalf("find forged oauth: %v", err)
	}
	if oauth != nil {
		t.Fatalf("generic oauth created mini association: %#v", oauth)
	}
}
