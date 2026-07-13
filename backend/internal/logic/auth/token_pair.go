package auth

import (
	"errors"

	"chasing_points/internal/pkg"
	"chasing_points/internal/svc"

	jwt "github.com/golang-jwt/jwt/v5"
)

const (
	accessTokenType  = "access"
	refreshTokenType = "refresh"
)

type authTokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

func issueAuthTokenPair(userId int64, svcCtx *svc.ServiceContext) (*authTokenPair, error) {
	if svcCtx == nil || svcCtx.Config.Auth.AccessSecret == "" {
		return nil, errors.New("auth config missing")
	}

	accessToken, err := pkg.GenerateTypedToken(userId, svcCtx.Config.Auth.AccessSecret, svcCtx.Config.Auth.AccessExpire, accessTokenType)
	if err != nil {
		return nil, err
	}

	refreshToken, err := pkg.GenerateTypedToken(userId, svcCtx.Config.Auth.AccessSecret, resolveRefreshExpire(svcCtx), refreshTokenType)
	if err != nil {
		return nil, err
	}

	return &authTokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    svcCtx.Config.Auth.AccessExpire,
	}, nil
}

func resolveRefreshExpire(svcCtx *svc.ServiceContext) int64 {
	if svcCtx != nil && svcCtx.Config.Auth.RefreshExpire > 0 {
		return svcCtx.Config.Auth.RefreshExpire
	}
	return 30 * 24 * 3600
}

func parseRefreshTokenUserId(tokenString string, secret string) (int64, error) {
	claims := &pkg.JwtClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return 0, err
	}
	if !token.Valid || claims.UserId <= 0 {
		return 0, jwt.ErrTokenInvalidClaims
	}
	// 兼容旧版登录态：历史 refresh_token 没有 token_type，只能在自然过期前续一次。
	if claims.TokenType != "" && claims.TokenType != refreshTokenType {
		return 0, jwt.ErrTokenInvalidClaims
	}
	return claims.UserId, nil
}
