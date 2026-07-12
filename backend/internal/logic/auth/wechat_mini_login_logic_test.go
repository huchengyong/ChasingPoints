package auth

import (
	"context"
	"testing"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/pkg/wechatmini"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeWechatMiniClient struct {
	identity *wechatmini.Identity
	phone    string
	err      error
}

func (f *fakeWechatMiniClient) ExchangeLoginCode(context.Context, string) (*wechatmini.Identity, error) {
	return f.identity, f.err
}

func (f *fakeWechatMiniClient) GetPhoneNumber(context.Context, string) (string, error) {
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

	cfg := config.Config{}
	cfg.Auth.AccessSecret = "wechat-mini-test-secret"
	cfg.Auth.AccessExpire = 3600
	cfg.Auth.RefreshExpire = 7200

	return &svc.ServiceContext{
		DB:               db,
		Config:           cfg,
		UserModel:        model.NewUserModel(db),
		OauthModel:       model.NewUserOauthModel(db),
		WechatMiniClient: client,
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
	if !resp.Success || !resp.NeedBindPhone || resp.UserInfo == nil || resp.UserInfo.Nickname != "微信用户" {
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
	if !resp.Success || resp.UserInfo == nil || resp.UserInfo.Id != existingUser.Id {
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
		Provider: wechatMiniProvider,
		OpenId:   "forged-openid",
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
