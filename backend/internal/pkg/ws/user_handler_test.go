package ws

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	pkgx "chasing_points/internal/pkg"
	"chasing_points/internal/svc"

	"github.com/gorilla/websocket"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newUserWSHandlerTestServer(t *testing.T) (*httptest.Server, *Hub, string) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("prepare user ws schema: %v", err)
	}
	for _, user := range []model.User{
		{Id: 1001, Nickname: "有效用户", Status: 1},
		{Id: 1002, Nickname: "停用用户", Status: 1},
	} {
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("create user %d: %v", user.Id, err)
		}
	}
	if err := db.Model(&model.User{}).Where("id = ?", 1002).Update("status", 0).Error; err != nil {
		t.Fatalf("disable user: %v", err)
	}

	const secret = "user-ws-test-secret"
	cfg := config.Config{}
	cfg.Auth.AccessSecret = secret
	hub := NewHub()
	previousHub := GlobalHub
	GlobalHub = hub
	t.Cleanup(func() { GlobalHub = previousHub })
	server := httptest.NewServer(UserWSHandler(&svc.ServiceContext{
		Config:    cfg,
		DB:        db,
		UserModel: model.NewUserModel(db),
	}))
	t.Cleanup(server.Close)
	return server, hub, secret
}

func dialUserWS(t *testing.T, server *httptest.Server, token string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/?token=" + token
	return websocket.DefaultDialer.Dial(url, nil)
}

func TestUserWSHandlerAllowsActiveAccessAndLegacyTokens(t *testing.T) {
	server, hub, secret := newUserWSHandlerTestServer(t)
	accessToken, err := pkgx.GenerateTypedToken(1001, secret, 60, pkgx.AccessTokenType)
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}
	legacyToken, err := pkgx.GenerateToken(1001, secret, 60)
	if err != nil {
		t.Fatalf("generate legacy token: %v", err)
	}

	for _, token := range []string{accessToken, legacyToken} {
		conn, response, err := dialUserWS(t, server, token)
		if err != nil || response == nil || response.StatusCode != http.StatusSwitchingProtocols {
			t.Fatalf("active user connection failed: response=%v err=%v", response, err)
		}
		conn.Close()
		select {
		case client := <-hub.RegisterUser:
			if client.UserId != 1001 {
				t.Fatalf("unexpected registered user: %+v", client)
			}
		case <-time.After(time.Second):
			t.Fatal("active user connection was not registered")
		}
	}
}

func TestUserWSHandlerRejectsRefreshDeletedDisabledAndAdminTokens(t *testing.T) {
	server, hub, secret := newUserWSHandlerTestServer(t)
	refreshToken, _ := pkgx.GenerateTypedToken(1001, secret, 60, pkgx.RefreshTokenType)
	deletedUserToken, _ := pkgx.GenerateTypedToken(9999, secret, 60, pkgx.AccessTokenType)
	disabledUserToken, _ := pkgx.GenerateTypedToken(1002, secret, 60, pkgx.AccessTokenType)
	adminToken, _ := pkgx.GenerateToken(-1, secret, 60)

	for name, token := range map[string]string{
		"refresh token":  refreshToken,
		"deleted user":   deletedUserToken,
		"disabled user":  disabledUserToken,
		"admin identity": adminToken,
	} {
		t.Run(name, func(t *testing.T) {
			conn, response, err := dialUserWS(t, server, token)
			if conn != nil {
				conn.Close()
			}
			if err == nil || response == nil || response.StatusCode != http.StatusUnauthorized {
				t.Fatalf("expected 401, response=%v err=%v", response, err)
			}
			select {
			case client := <-hub.RegisterUser:
				t.Fatalf("rejected connection registered client: %+v", client)
			default:
			}
		})
	}
}
