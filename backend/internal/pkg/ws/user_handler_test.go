package ws

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/pkg/wsticket"
	"chasing_points/internal/svc"

	"github.com/gorilla/websocket"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const userWSAllowedOrigin = "http://user-ws.test"

func newUserWSHandlerTestServer(t *testing.T) (*httptest.Server, *Hub, *fakeHandlerWSTicketStore) {
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

	cfg := config.Config{}
	cfg.Security.WebSocketAllowedOrigins = userWSAllowedOrigin
	hub := NewHub()
	ticketStore := newFakeHandlerWSTicketStore()
	previousHub := GlobalHub
	GlobalHub = hub
	t.Cleanup(func() { GlobalHub = previousHub })
	server := httptest.NewServer(UserWSHandler(&svc.ServiceContext{
		Config:        cfg,
		DB:            db,
		UserModel:     model.NewUserModel(db),
		WSTicketStore: ticketStore,
	}))
	t.Cleanup(server.Close)
	return server, hub, ticketStore
}

func dialUserWS(t *testing.T, server *httptest.Server, ticket string) (*websocket.Conn, *http.Response, error) {
	return dialUserWSPathWithOrigin(t, server, "/?ticket="+ticket, userWSAllowedOrigin)
}

func dialUserWSPathWithOrigin(t *testing.T, server *httptest.Server, path string, origin string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	url := "ws" + strings.TrimPrefix(server.URL, "http") + path
	header := http.Header{}
	if origin != "" {
		header.Set("Origin", origin)
	}
	return websocket.DefaultDialer.Dial(url, header)
}

func TestUserWSHandlerAllowsActiveTicketOnce(t *testing.T) {
	server, hub, ticketStore := newUserWSHandlerTestServer(t)
	ticketStore.Add("user-ticket", wsticket.Claims{UserID: 1001, Scope: wsticket.ScopeUser})
	conn, response, err := dialUserWS(t, server, "user-ticket")
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

	conn, response, err = dialUserWS(t, server, "user-ticket")
	if conn != nil {
		conn.Close()
	}
	if err == nil || response == nil || response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("replayed ticket should be rejected, response=%v err=%v", response, err)
	}
}

func TestUserWSHandlerRejectsDisallowedOriginBeforeRegistration(t *testing.T) {
	server, hub, ticketStore := newUserWSHandlerTestServer(t)
	ticketStore.Add("origin-ticket", wsticket.Claims{UserID: 1001, Scope: wsticket.ScopeUser})
	conn, response, err := dialUserWSPathWithOrigin(t, server, "/?ticket=origin-ticket", "http://evil.test")
	if conn != nil {
		conn.Close()
	}
	if err == nil || response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("expected origin rejection, response=%v err=%v", response, err)
	}
	select {
	case client := <-hub.RegisterUser:
		t.Fatalf("origin rejected connection registered client: %+v", client)
	default:
	}
}

func TestUserWSHandlerRejectsQueryTokenInvalidDeletedDisabledAndAdminTickets(t *testing.T) {
	server, hub, ticketStore := newUserWSHandlerTestServer(t)
	ticketStore.Add("deleted-user", wsticket.Claims{UserID: 9999, Scope: wsticket.ScopeUser})
	ticketStore.Add("disabled-user", wsticket.Claims{UserID: 1002, Scope: wsticket.ScopeUser})
	ticketStore.Add("admin-identity", wsticket.Claims{UserID: -1, Scope: wsticket.ScopeUser})

	for name, path := range map[string]string{
		"old query token": "/?token=old-access-token",
		"invalid ticket":  "/?ticket=missing",
		"expired ticket":  "/?ticket=expired-ticket",
		"deleted user":    "/?ticket=deleted-user",
		"disabled user":   "/?ticket=disabled-user",
		"admin identity":  "/?ticket=admin-identity",
	} {
		t.Run(name, func(t *testing.T) {
			conn, response, err := dialUserWSPathWithOrigin(t, server, path, userWSAllowedOrigin)
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
