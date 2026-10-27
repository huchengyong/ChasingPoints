package logic

import (
	"context"
	"errors"
	"strings"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg/matchinvite"
	"chasing_points/internal/svc"
)

const (
	// MatchInviteExpiredMessage 统一用于短码失效、token 过期和邀请人不可用，不暴露具体原因。
	MatchInviteExpiredMessage     = "匹配二维码已失效，请让对方刷新二维码"
	MatchInviteUnavailableMessage = "匹配二维码服务暂不可用"
)

// NewMatchInviteSigner 按配置创建匹配邀请签名器；签发、校验与短码兑换共用。
func NewMatchInviteSigner(svcCtx *svc.ServiceContext) (*matchinvite.Signer, error) {
	if svcCtx == nil {
		return nil, matchinvite.ErrSignerConfig
	}
	return matchinvite.NewSigner(
		svcCtx.Config.Security.MatchInvite.SigningSecret,
		time.Duration(svcCtx.Config.Security.MatchInvite.TTLSeconds)*time.Second,
	)
}

// VerifyMatchInvite 校验签名 token，失败时返回面向用户的提示。
func VerifyMatchInvite(svcCtx *svc.ServiceContext, token string) (*matchinvite.Claims, string) {
	signer, err := NewMatchInviteSigner(svcCtx)
	if err != nil {
		return nil, MatchInviteUnavailableMessage
	}
	claims, err := signer.Verify(strings.TrimSpace(token))
	if err == nil {
		return claims, ""
	}
	if errors.Is(err, matchinvite.ErrExpiredToken) {
		return nil, MatchInviteExpiredMessage
	}
	return nil, "无效的匹配二维码"
}

// MatchInviteCodeResult 是短码兑换后的可信结果，供登录态兑换与匿名预览共同使用。
type MatchInviteCodeResult struct {
	Token     string
	ExpiresAt time.Time
	Inviter   *model.User
}

func ResolveMatchInviteCode(ctx context.Context, svcCtx *svc.ServiceContext, code string) (*MatchInviteCodeResult, string) {
	if svcCtx == nil || svcCtx.MatchInviteCodeStore == nil || svcCtx.UserModel == nil {
		return nil, MatchInviteUnavailableMessage
	}
	token, err := svcCtx.MatchInviteCodeStore.Resolve(ctx, strings.TrimSpace(code))
	if err != nil {
		if errors.Is(err, matchinvite.ErrCodeNotFound) || errors.Is(err, matchinvite.ErrInvalidCode) {
			return nil, MatchInviteExpiredMessage
		}
		return nil, MatchInviteUnavailableMessage
	}

	// 短码入口不区分签名无效与过期，统一按失效提示。
	claims, message := VerifyMatchInvite(svcCtx, token)
	if message == MatchInviteUnavailableMessage {
		return nil, message
	}
	if message != "" {
		return nil, MatchInviteExpiredMessage
	}

	inviter, err := svcCtx.UserModel.FindByIdWithContext(ctx, claims.InviterUserID)
	if err != nil {
		return nil, MatchInviteUnavailableMessage
	}
	if inviter == nil || inviter.Status != 1 {
		return nil, MatchInviteExpiredMessage
	}

	return &MatchInviteCodeResult{
		Token:     token,
		ExpiresAt: time.Unix(claims.ExpiresAt, 0).UTC(),
		Inviter:   inviter,
	}, ""
}
