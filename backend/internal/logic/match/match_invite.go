package match

import (
	"errors"
	"strings"
	"time"

	"chasing_points/internal/pkg/matchinvite"
	"chasing_points/internal/svc"
)

func newMatchInviteSigner(svcCtx *svc.ServiceContext) (*matchinvite.Signer, error) {
	if svcCtx == nil {
		return nil, matchinvite.ErrSignerConfig
	}
	return matchinvite.NewSigner(
		svcCtx.Config.Security.MatchInvite.SigningSecret,
		time.Duration(svcCtx.Config.Security.MatchInvite.TTLSeconds)*time.Second,
	)
}

func verifyMatchInvite(svcCtx *svc.ServiceContext, token string) (*matchinvite.Claims, string) {
	signer, err := newMatchInviteSigner(svcCtx)
	if err != nil {
		return nil, "匹配二维码服务暂不可用"
	}
	claims, err := signer.Verify(strings.TrimSpace(token))
	if err == nil {
		return claims, ""
	}
	if errors.Is(err, matchinvite.ErrExpiredToken) {
		return nil, "匹配二维码已失效，请让对方刷新二维码"
	}
	return nil, "无效的匹配二维码"
}
