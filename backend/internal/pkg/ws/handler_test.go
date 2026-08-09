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

func newMatchWSHandlerTestServer(t *testing.T) (*httptest.Server, *Hub, string) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}, &model.MatchRound{}, &model.MatchAction{}); err != nil {
		t.Fatalf("prepare ws handler schema: %v", err)
	}
	opponentID := int64(2002)
	refereeID := int64(3003)
	for _, user := range []model.User{
		{Id: 1001, Nickname: "发起者", Status: 1},
		{Id: 2002, Nickname: "对手", Status: 1},
		{Id: 3003, Nickname: "裁判", Status: 1},
		{Id: 4004, Nickname: "非参与者", Status: 1},
		{Id: 6006, Nickname: "停用用户", Status: 1},
	} {
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("create user %d: %v", user.Id, err)
		}
	}
	if err := db.Model(&model.User{}).Where("id = ?", 6006).Update("status", 0).Error; err != nil {
		t.Fatalf("disable ws test user: %v", err)
	}
	for _, match := range []model.Match{
		{Id: 1, UserId: 1001, OpponentId: &opponentID, OpponentName: "对手", GameType: 3, MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic, Status: 1, CurrentFrameStarted: true, MatchTime: time.Now()},
		{Id: 2, UserId: 1001, OpponentId: &opponentID, OpponentName: "对手", GameType: 3, MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate, RefereeUserId: &refereeID, Status: 1, CurrentFrameStarted: true, MatchTime: time.Now()},
	} {
		if err := db.Create(&match).Error; err != nil {
			t.Fatalf("create match %d: %v", match.Id, err)
		}
	}
	const secret = "match-ws-test-secret"
	cfg := config.Config{}
	cfg.Auth.AccessSecret = secret
	hub := NewHub()
	previousHub := GlobalHub
	GlobalHub = hub
	t.Cleanup(func() { GlobalHub = previousHub })
	server := httptest.NewServer(MatchWSHandler(&svc.ServiceContext{
		Config:     cfg,
		DB:         db,
		MatchModel: model.NewMatchModel(db),
		UserModel:  model.NewUserModel(db),
	}))
	t.Cleanup(server.Close)
	return server, hub, secret
}

func dialMatchWS(t *testing.T, server *httptest.Server, path string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	url := "ws" + strings.TrimPrefix(server.URL, "http") + path
	return websocket.DefaultDialer.Dial(url, nil)
}

func TestMatchWSHandlerAuthorizesBeforeRegistration(t *testing.T) {
	server, hub, secret := newMatchWSHandlerTestServer(t)
	publicConn, response, err := dialMatchWS(t, server, "/?match_id=1")
	if err != nil || response == nil || response.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("public anonymous connection failed: status=%v err=%v", response, err)
	}
	defer publicConn.Close()
	select {
	case client := <-hub.Register:
		if client.UserId != 0 || client.MatchId != 1 {
			t.Fatalf("unexpected public anonymous registration: %+v", client)
		}
	case <-time.After(time.Second):
		t.Fatal("public anonymous connection was not registered")
	}

	for _, userID := range []int64{1001, 2002, 3003} {
		token, err := pkgx.GenerateToken(userID, secret, 60)
		if err != nil {
			t.Fatalf("generate token: %v", err)
		}
		conn, response, err := dialMatchWS(t, server, "/?match_id=2&token="+token)
		if err != nil || response == nil || response.StatusCode != http.StatusSwitchingProtocols {
			t.Fatalf("private participant %d connection failed: status=%v err=%v", userID, response, err)
		}
		conn.Close()
		select {
		case client := <-hub.Register:
			if client.UserId != userID || client.MatchId != 2 {
				t.Fatalf("unexpected private registration: %+v", client)
			}
		case <-time.After(time.Second):
			t.Fatalf("private participant %d was not registered", userID)
		}
	}
}

func TestMatchWSHandlerRejectsUnauthorizedHandshakesWithoutRegistration(t *testing.T) {
	server, hub, secret := newMatchWSHandlerTestServer(t)
	nonParticipantToken, _ := pkgx.GenerateToken(4004, secret, 60)
	refreshToken, _ := pkgx.GenerateTypedToken(1001, secret, 60, pkgx.RefreshTokenType)
	deletedUserToken, _ := pkgx.GenerateTypedToken(9999, secret, 60, pkgx.AccessTokenType)
	disabledUserToken, _ := pkgx.GenerateTypedToken(6006, secret, 60, pkgx.AccessTokenType)
	tests := []struct {
		name       string
		path       string
		statusCode int
	}{
		{name: "match not found", path: "/?match_id=999", statusCode: http.StatusNotFound},
		{name: "invalid token", path: "/?match_id=1&token=invalid", statusCode: http.StatusUnauthorized},
		{name: "refresh token", path: "/?match_id=1&token=" + refreshToken, statusCode: http.StatusUnauthorized},
		{name: "deleted user token", path: "/?match_id=1&token=" + deletedUserToken, statusCode: http.StatusUnauthorized},
		{name: "disabled user token", path: "/?match_id=1&token=" + disabledUserToken, statusCode: http.StatusUnauthorized},
		{name: "private anonymous", path: "/?match_id=2", statusCode: http.StatusUnauthorized},
		{name: "private non participant", path: "/?match_id=2&token=" + nonParticipantToken, statusCode: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, response, err := dialMatchWS(t, server, tt.path)
			if conn != nil {
				conn.Close()
			}
			if err == nil || response == nil || response.StatusCode != tt.statusCode {
				t.Fatalf("expected status %d, response=%v err=%v", tt.statusCode, response, err)
			}
			select {
			case client := <-hub.Register:
				t.Fatalf("rejected connection registered client: %+v", client)
			default:
			}
		})
	}
}
