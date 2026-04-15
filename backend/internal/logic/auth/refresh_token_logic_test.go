package auth

import (
	"context"
	"testing"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/pkg"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	jwt "github.com/golang-jwt/jwt/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newRefreshTokenTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("prepare refresh token schema: %v", err)
	}

	cfg := config.Config{}
	cfg.Auth.AccessSecret = "refresh-token-secret"
	cfg.Auth.AccessExpire = 7 * 24 * 3600
	cfg.Auth.RefreshExpire = 30 * 24 * 3600

	return &svc.ServiceContext{
		DB:        db,
		Config:    cfg,
		UserModel: model.NewUserModel(db),
	}
}

func parseRefreshTestClaims(t *testing.T, tokenString string, secret string) jwt.MapClaims {
	t.Helper()

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		t.Fatalf("expected valid jwt map claims")
	}
	return claims
}

func TestRefreshTokenIssuesNewAccessAndRefreshPair(t *testing.T) {
	svcCtx := newRefreshTokenTestSvc(t)
	if err := svcCtx.UserModel.Create(&model.User{Id: 1001, Nickname: "续期用户", Status: 1}); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	refreshToken, err := pkg.GenerateTypedToken(1001, svcCtx.Config.Auth.AccessSecret, svcCtx.Config.Auth.RefreshExpire, "refresh")
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}

	logic := NewRefreshTokenLogic(context.Background(), svcCtx)
	resp, err := logic.RefreshToken(&types.RefreshTokenReq{RefreshToken: refreshToken})
	if err != nil {
		t.Fatalf("refresh token: %v", err)
	}
	if resp == nil || !resp.Success {
		t.Fatalf("expected successful refresh response, got %#v", resp)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatalf("expected refreshed token pair, got %#v", resp)
	}
	if resp.ExpiresIn != svcCtx.Config.Auth.AccessExpire {
		t.Fatalf("expected access expiry %d, got %d", svcCtx.Config.Auth.AccessExpire, resp.ExpiresIn)
	}

	accessClaims := parseRefreshTestClaims(t, resp.AccessToken, svcCtx.Config.Auth.AccessSecret)
	if accessClaims["token_type"] != "access" || int64(accessClaims["user_id"].(float64)) != 1001 {
		t.Fatalf("unexpected access claims: %#v", accessClaims)
	}
	refreshClaims := parseRefreshTestClaims(t, resp.RefreshToken, svcCtx.Config.Auth.AccessSecret)
	if refreshClaims["token_type"] != "refresh" || int64(refreshClaims["user_id"].(float64)) != 1001 {
		t.Fatalf("unexpected refresh claims: %#v", refreshClaims)
	}
}

func TestRefreshTokenRejectsAccessToken(t *testing.T) {
	svcCtx := newRefreshTokenTestSvc(t)
	if err := svcCtx.UserModel.Create(&model.User{Id: 1002, Nickname: "访问令牌用户", Status: 1}); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	accessToken, err := pkg.GenerateTypedToken(1002, svcCtx.Config.Auth.AccessSecret, svcCtx.Config.Auth.AccessExpire, "access")
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}

	logic := NewRefreshTokenLogic(context.Background(), svcCtx)
	resp, err := logic.RefreshToken(&types.RefreshTokenReq{RefreshToken: accessToken})
	if err != nil {
		t.Fatalf("reject access token should be business response, got error: %v", err)
	}
	if resp == nil || resp.Success {
		t.Fatalf("expected access token to be rejected as refresh token, got %#v", resp)
	}
}

func TestRefreshTokenAcceptsLegacyUntypedRefreshToken(t *testing.T) {
	svcCtx := newRefreshTokenTestSvc(t)
	if err := svcCtx.UserModel.Create(&model.User{Id: 1003, Nickname: "旧版令牌用户", Status: 1}); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	legacyRefreshToken, err := pkg.GenerateToken(1003, svcCtx.Config.Auth.AccessSecret, svcCtx.Config.Auth.RefreshExpire)
	if err != nil {
		t.Fatalf("generate legacy refresh token: %v", err)
	}

	logic := NewRefreshTokenLogic(context.Background(), svcCtx)
	resp, err := logic.RefreshToken(&types.RefreshTokenReq{RefreshToken: legacyRefreshToken})
	if err != nil {
		t.Fatalf("refresh legacy token: %v", err)
	}
	if resp == nil || !resp.Success || resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatalf("expected legacy refresh token to be accepted once, got %#v", resp)
	}
}
