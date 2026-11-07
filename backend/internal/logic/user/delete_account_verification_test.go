package user

import (
	"context"
	"errors"
	"testing"

	"chasing_points/internal/model"
	oauthverify "chasing_points/internal/pkg/oauth"
	"chasing_points/internal/pkg/wechatmini"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type deleteAccountVerifier struct {
	identity *oauthverify.Identity
	err      error
	request  oauthverify.VerifyRequest
	called   bool
}

func (v *deleteAccountVerifier) Verify(_ context.Context, req oauthverify.VerifyRequest) (*oauthverify.Identity, error) {
	v.called = true
	v.request = req
	if v.err != nil {
		return nil, v.err
	}
	return v.identity, nil
}

type deleteAccountWechatClient struct {
	identity *wechatmini.Identity
	err      error
}

func (c deleteAccountWechatClient) ExchangeLoginCode(_ context.Context, _ string) (*wechatmini.Identity, error) {
	return c.identity, c.err
}

func (c deleteAccountWechatClient) GetPhoneNumber(_ context.Context, _ string) (string, error) {
	return "", nil
}

func newDeleteAccountVerificationTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.Exec(`
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
		t.Fatalf("prepare oauth schema: %v", err)
	}
	return &svc.ServiceContext{OauthModel: model.NewUserOauthModel(db)}
}

func TestOAuthDeletionReauthUsesFreshProviderCredentialAndBoundIdentity(t *testing.T) {
	svcCtx := newDeleteAccountVerificationTestSvc(t)
	if err := svcCtx.OauthModel.Create(&model.UserOauth{UserId: 101, Provider: "huawei", OpenId: "verified-subject"}); err != nil {
		t.Fatalf("create oauth association: %v", err)
	}
	verifier := &deleteAccountVerifier{identity: &oauthverify.Identity{Provider: "huawei", Subject: "verified-subject"}}
	svcCtx.OAuthVerifier = verifier

	_, err := NewDeleteAccountLogic(context.Background(), svcCtx).verifyOAuthDeletionCredential(101, &types.DeleteAccountReq{
		Provider:       "huawei",
		Credential:     "fresh-provider-credential",
		CredentialType: "authorization_code",
		Platform:       "app-plus",
	})
	if err != nil {
		t.Fatalf("verify oauth reauthentication: %v", err)
	}
	if !verifier.called || verifier.request.Credential != "fresh-provider-credential" || verifier.request.Provider != "huawei" {
		t.Fatalf("unexpected verifier request: %+v", verifier.request)
	}
}

func TestOAuthDeletionReauthRejectsSessionOnlyAndUnboundIdentity(t *testing.T) {
	svcCtx := newDeleteAccountVerificationTestSvc(t)
	verifier := &deleteAccountVerifier{identity: &oauthverify.Identity{Provider: "huawei", Subject: "other-subject"}}
	svcCtx.OAuthVerifier = verifier

	logic := NewDeleteAccountLogic(context.Background(), svcCtx)
	if _, err := logic.verifyOAuthDeletionCredential(101, &types.DeleteAccountReq{Provider: "huawei"}); !errors.Is(err, errDeleteAccountOAuthVerificationRequired) {
		t.Fatalf("session-only request error = %v, want credential-required", err)
	}
	if verifier.called {
		t.Fatal("session-only request must not call the provider verifier")
	}

	if _, err := logic.verifyOAuthDeletionCredential(101, &types.DeleteAccountReq{Provider: "huawei", Credential: "fresh-credential"}); !errors.Is(err, errDeleteAccountOAuthVerificationFailed) {
		t.Fatalf("unbound provider identity error = %v, want verification failure", err)
	}
}

func TestOAuthDeletionReauthRejectsProviderFailure(t *testing.T) {
	svcCtx := newDeleteAccountVerificationTestSvc(t)
	svcCtx.OAuthVerifier = &deleteAccountVerifier{err: oauthverify.NewVerifyError(oauthverify.ErrorRejected, "credential expired", nil)}

	_, err := NewDeleteAccountLogic(context.Background(), svcCtx).verifyOAuthDeletionCredential(101, &types.DeleteAccountReq{
		Provider:   "huawei",
		Credential: "expired-provider-credential",
	})
	if !errors.Is(err, errDeleteAccountOAuthVerificationFailed) {
		t.Fatalf("rejected credential error = %v, want verification failure", err)
	}
}

func TestOAuthDeletionReauthAcceptsFreshWechatMiniCodeForBoundAccount(t *testing.T) {
	svcCtx := newDeleteAccountVerificationTestSvc(t)
	if err := svcCtx.OauthModel.Create(&model.UserOauth{UserId: 101, Provider: wechatMiniDeletionProvider, OpenId: "wechat-subject"}); err != nil {
		t.Fatalf("create wechat oauth association: %v", err)
	}
	svcCtx.WechatMiniClient = deleteAccountWechatClient{identity: &wechatmini.Identity{OpenID: "wechat-subject"}}

	identity, err := NewDeleteAccountLogic(context.Background(), svcCtx).verifyOAuthDeletionCredential(101, &types.DeleteAccountReq{
		Provider:   wechatMiniDeletionProvider,
		Credential: "fresh-wechat-login-code",
	})
	if err != nil || identity == nil || identity.Provider != wechatMiniDeletionProvider || identity.Subject != "wechat-subject" {
		t.Fatalf("verify wechat deletion reauthentication: identity=%#v err=%v", identity, err)
	}
}
