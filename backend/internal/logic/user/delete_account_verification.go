package user

import (
	"errors"
	"strings"

	oauthverify "chasing_points/internal/pkg/oauth"
	"chasing_points/internal/pkg/wechatmini"
	"chasing_points/internal/types"
)

const wechatMiniDeletionProvider = "weixin_mini_program"

var (
	errDeleteAccountOAuthVerificationRequired = errors.New("请重新完成第三方身份验证")
	errDeleteAccountOAuthVerificationFailed   = errors.New("第三方身份验证失败")
	errDeleteAccountOAuthServiceUnavailable   = errors.New("第三方身份验证服务暂不可用")
)

func (l *DeleteAccountLogic) verifyOAuthDeletionCredential(userID int64, req *types.DeleteAccountReq) (*oauthverify.Identity, error) {
	if userID <= 0 || req == nil {
		return nil, errDeleteAccountOAuthVerificationRequired
	}
	if l == nil || l.svcCtx == nil || l.svcCtx.OauthModel == nil {
		return nil, errDeleteAccountOAuthServiceUnavailable
	}

	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	credential := strings.TrimSpace(req.Credential)
	if provider == "" || credential == "" {
		return nil, errDeleteAccountOAuthVerificationRequired
	}
	if provider == wechatMiniDeletionProvider {
		return l.verifyWechatMiniDeletionCredential(userID, credential)
	}
	if l.svcCtx.OAuthVerifier == nil {
		return nil, errDeleteAccountOAuthServiceUnavailable
	}

	identity, err := l.svcCtx.OAuthVerifier.Verify(l.ctx, oauthverify.VerifyRequest{
		Provider:       provider,
		Credential:     credential,
		CredentialType: strings.TrimSpace(req.CredentialType),
		Platform:       strings.TrimSpace(req.Platform),
	})
	if err != nil {
		if oauthverify.CategoryOf(err) == oauthverify.ErrorUnavailable {
			return nil, errDeleteAccountOAuthServiceUnavailable
		}
		return nil, errDeleteAccountOAuthVerificationFailed
	}
	if identity == nil || strings.ToLower(strings.TrimSpace(identity.Provider)) != provider || strings.TrimSpace(identity.Subject) == "" {
		return nil, errDeleteAccountOAuthVerificationFailed
	}

	oauth, err := l.svcCtx.OauthModel.FindByProviderAndOpenId(provider, strings.TrimSpace(identity.Subject))
	if err != nil {
		return nil, errDeleteAccountOAuthServiceUnavailable
	}
	if oauth == nil || oauth.UserId != userID {
		return nil, errDeleteAccountOAuthVerificationFailed
	}
	return identity, nil
}

func (l *DeleteAccountLogic) verifyWechatMiniDeletionCredential(userID int64, credential string) (*oauthverify.Identity, error) {
	if l.svcCtx.WechatMiniClient == nil {
		return nil, errDeleteAccountOAuthServiceUnavailable
	}
	identity, err := l.svcCtx.WechatMiniClient.ExchangeLoginCode(l.ctx, credential)
	if err != nil {
		if wechatmini.IsErrorKind(err, wechatmini.ErrorKindRejected) {
			return nil, errDeleteAccountOAuthVerificationFailed
		}
		return nil, errDeleteAccountOAuthServiceUnavailable
	}
	if identity == nil || strings.TrimSpace(identity.OpenID) == "" {
		return nil, errDeleteAccountOAuthVerificationFailed
	}

	oauth, err := l.svcCtx.OauthModel.FindByProviderAndOpenId(wechatMiniDeletionProvider, strings.TrimSpace(identity.OpenID))
	if err != nil {
		return nil, errDeleteAccountOAuthServiceUnavailable
	}
	if oauth == nil || oauth.UserId != userID {
		return nil, errDeleteAccountOAuthVerificationFailed
	}
	return &oauthverify.Identity{
		Provider: wechatMiniDeletionProvider,
		Subject:  strings.TrimSpace(identity.OpenID),
		UnionID:  strings.TrimSpace(identity.UnionID),
	}, nil
}
