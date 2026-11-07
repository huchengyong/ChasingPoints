package ws

import (
	"context"
	"fmt"
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

const matchWSAllowedOrigin = "http://ws.test"

func newMatchWSHandlerTestServer(t *testing.T) (*httptest.Server, *Hub, *fakeHandlerWSTicketStore) {
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
	cfg := config.Config{}
	cfg.Security.WebSocketAllowedOrigins = matchWSAllowedOrigin
	hub := NewHub()
	ticketStore := newFakeHandlerWSTicketStore()
	previousHub := GlobalHub
	GlobalHub = hub
	t.Cleanup(func() { GlobalHub = previousHub })
	server := httptest.NewServer(MatchWSHandler(&svc.ServiceContext{
		Config:        cfg,
		DB:            db,
		MatchModel:    model.NewMatchModel(db),
		UserModel:     model.NewUserModel(db),
		WSTicketStore: ticketStore,
	}))
	t.Cleanup(server.Close)
	return server, hub, ticketStore
}

func dialMatchWS(t *testing.T, server *httptest.Server, path string) (*websocket.Conn, *http.Response, error) {
	return dialMatchWSWithOrigin(t, server, path, matchWSAllowedOrigin)
}

func dialMatchWSWithOrigin(t *testing.T, server *httptest.Server, path string, origin string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	url := "ws" + strings.TrimPrefix(server.URL, "http") + path
	header := http.Header{}
	if origin != "" {
		header.Set("Origin", origin)
	}
	return websocket.DefaultDialer.Dial(url, header)
}

func TestMatchWSHandlerAuthorizesBeforeRegistration(t *testing.T) {
	server, hub, ticketStore := newMatchWSHandlerTestServer(t)
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
		ticket := fmt.Sprintf("private-ticket-%d", userID)
		ticketStore.Add(ticket, wsticket.Claims{UserID: userID, Scope: wsticket.ScopeMatch, MatchID: 2})
		conn, response, err := dialMatchWS(t, server, "/?match_id=2&ticket="+ticket)
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

		conn, response, err = dialMatchWS(t, server, "/?match_id=2&ticket="+ticket)
		if conn != nil {
			conn.Close()
		}
		if err == nil || response == nil || response.StatusCode != http.StatusUnauthorized {
			t.Fatalf("replayed private participant %d ticket should fail: status=%v err=%v", userID, response, err)
		}
	}
}

func TestMatchWSHandlerRejectsUnauthorizedHandshakesWithoutRegistration(t *testing.T) {
	server, hub, ticketStore := newMatchWSHandlerTestServer(t)
	ticketStore.Add("non-participant", wsticket.Claims{UserID: 4004, Scope: wsticket.ScopeMatch, MatchID: 2})
	ticketStore.Add("deleted-user", wsticket.Claims{UserID: 9999, Scope: wsticket.ScopeMatch, MatchID: 1})
	ticketStore.Add("disabled-user", wsticket.Claims{UserID: 6006, Scope: wsticket.ScopeMatch, MatchID: 1})
	ticketStore.Add("wrong-match", wsticket.Claims{UserID: 1001, Scope: wsticket.ScopeMatch, MatchID: 1})
	tests := []struct {
		name       string
		path       string
		statusCode int
	}{
		{name: "match not found", path: "/?match_id=999", statusCode: http.StatusNotFound},
		{name: "old query token", path: "/?match_id=1&token=old-access-token", statusCode: http.StatusUnauthorized},
		{name: "invalid ticket", path: "/?match_id=1&ticket=missing", statusCode: http.StatusUnauthorized},
		{name: "expired ticket", path: "/?match_id=1&ticket=expired-ticket", statusCode: http.StatusUnauthorized},
		{name: "deleted logout user ticket", path: "/?match_id=1&ticket=deleted-user", statusCode: http.StatusUnauthorized},
		{name: "disabled user ticket", path: "/?match_id=1&ticket=disabled-user", statusCode: http.StatusUnauthorized},
		{name: "wrong match ticket", path: "/?match_id=2&ticket=wrong-match", statusCode: http.StatusUnauthorized},
		{name: "private anonymous", path: "/?match_id=2", statusCode: http.StatusUnauthorized},
		{name: "private non participant", path: "/?match_id=2&ticket=non-participant", statusCode: http.StatusForbidden},
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

func TestMatchWSHandlerRejectsDisallowedOriginBeforeRegistration(t *testing.T) {
	server, hub, _ := newMatchWSHandlerTestServer(t)
	conn, response, err := dialMatchWSWithOrigin(t, server, "/?match_id=1", "http://evil.test")
	if conn != nil {
		conn.Close()
	}
	if err == nil || response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("expected origin rejection, response=%v err=%v", response, err)
	}
	select {
	case client := <-hub.Register:
		t.Fatalf("origin rejected connection registered client: %+v", client)
	default:
	}
}

type fakeHandlerWSTicketStore struct {
	claims   map[string]wsticket.Claims
	consumed map[string]bool
}

func newFakeHandlerWSTicketStore() *fakeHandlerWSTicketStore {
	return &fakeHandlerWSTicketStore{
		claims:   map[string]wsticket.Claims{},
		consumed: map[string]bool{},
	}
}

func (s *fakeHandlerWSTicketStore) Add(ticket string, claims wsticket.Claims) {
	s.claims[ticket] = claims
}

func (s *fakeHandlerWSTicketStore) Issue(ctx context.Context, claims wsticket.Claims, ttl time.Duration) (string, time.Duration, error) {
	return "", 0, wsticket.ErrStoreMissing
}

func (s *fakeHandlerWSTicketStore) Consume(ctx context.Context, ticket string, expectedScope string) (*wsticket.Claims, error) {
	claims, ok := s.claims[ticket]
	if !ok || s.consumed[ticket] || claims.Scope != expectedScope {
		return nil, wsticket.ErrTicketNotFound
	}
	s.consumed[ticket] = true
	return &claims, nil
}
