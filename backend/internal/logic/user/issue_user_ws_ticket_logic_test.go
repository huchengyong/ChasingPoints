package user

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg"
	"chasing_points/internal/pkg/httperror"
	"chasing_points/internal/pkg/wsticket"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestIssueUserWSTicketSignsActiveAccessUser(t *testing.T) {
	svcCtx := newUserWSTicketTestSvc(t)
	seedUserWSTicketUser(t, svcCtx, &model.User{Id: 11, Nickname: "在线用户", Status: 1})
	store := &fakeWSTicketStore{ticket: "user-ticket", ttl: 25 * time.Second}
	svcCtx.WSTicketStore = store

	logic := NewIssueUserWSTicketLogic(userWSTicketCtx(11, pkg.AccessTokenType), svcCtx)
	resp, err := logic.IssueUserWSTicket()
	if err != nil {
		t.Fatalf("issue user ws ticket: %v", err)
	}
	if !resp.Success || resp.Ticket != "user-ticket" || resp.Scope != wsticket.ScopeUser || resp.ExpiresInSeconds != 25 {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if len(store.issued) != 1 || store.issued[0].UserID != 11 || store.issued[0].Scope != wsticket.ScopeUser {
		t.Fatalf("unexpected issued claims: %+v", store.issued)
	}
}

func TestIssueUserWSTicketRejectsRefreshToken(t *testing.T) {
	svcCtx := newUserWSTicketTestSvc(t)
	seedUserWSTicketUser(t, svcCtx, &model.User{Id: 12, Nickname: "刷新令牌用户", Status: 1})
	store := &fakeWSTicketStore{ticket: "unused", ttl: wsticket.DefaultTTL}
	svcCtx.WSTicketStore = store

	_, err := NewIssueUserWSTicketLogic(userWSTicketCtx(12, pkg.RefreshTokenType), svcCtx).IssueUserWSTicket()
	assertHTTPError(t, err, http.StatusUnauthorized, "INVALID_ACCESS_TOKEN")
	if len(store.issued) != 0 {
		t.Fatalf("refresh token must not issue ticket: %+v", store.issued)
	}
}

func TestIssueUserWSTicketRejectsDisabledUser(t *testing.T) {
	svcCtx := newUserWSTicketTestSvc(t)
	seedUserWSTicketUser(t, svcCtx, &model.User{Id: 13, Nickname: "停用用户", Status: 1})
	if err := svcCtx.UserModel.UpdateStatus(13, 0); err != nil {
		t.Fatalf("disable user: %v", err)
	}
	svcCtx.WSTicketStore = &fakeWSTicketStore{ticket: "unused", ttl: wsticket.DefaultTTL}

	_, err := NewIssueUserWSTicketLogic(userWSTicketCtx(13, pkg.AccessTokenType), svcCtx).IssueUserWSTicket()
	assertHTTPError(t, err, http.StatusUnauthorized, "SESSION_INVALID")
}

func TestIssueUserWSTicketReturnsUnavailableWhenStoreFails(t *testing.T) {
	svcCtx := newUserWSTicketTestSvc(t)
	seedUserWSTicketUser(t, svcCtx, &model.User{Id: 14, Nickname: "Redis故障用户", Status: 1})
	svcCtx.WSTicketStore = &fakeWSTicketStore{err: errors.New("redis down")}

	_, err := NewIssueUserWSTicketLogic(userWSTicketCtx(14, pkg.AccessTokenType), svcCtx).IssueUserWSTicket()
	assertHTTPError(t, err, http.StatusServiceUnavailable, "WS_TICKET_UNAVAILABLE")
}

type fakeWSTicketStore struct {
	ticket string
	ttl    time.Duration
	err    error
	issued []wsticket.Claims
}

func (s *fakeWSTicketStore) Issue(ctx context.Context, claims wsticket.Claims, ttl time.Duration) (string, time.Duration, error) {
	if s.err != nil {
		return "", 0, s.err
	}
	if s.ticket == "" {
		s.ticket = "fake-ticket"
	}
	if s.ttl <= 0 {
		s.ttl = wsticket.DefaultTTL
	}
	s.issued = append(s.issued, claims)
	return s.ticket, s.ttl, nil
}

func (s *fakeWSTicketStore) Consume(ctx context.Context, ticket string, expectedScope string) (*wsticket.Claims, error) {
	return nil, wsticket.ErrTicketNotFound
}

func newUserWSTicketTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("prepare user schema: %v", err)
	}
	return &svc.ServiceContext{
		DB:        db,
		UserModel: model.NewUserModel(db),
	}
}

func seedUserWSTicketUser(t *testing.T, svcCtx *svc.ServiceContext, user *model.User) {
	t.Helper()
	if err := svcCtx.UserModel.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
}

func userWSTicketCtx(userID int64, tokenType string) context.Context {
	ctx := context.WithValue(context.Background(), "user_id", userID)
	if tokenType != "" {
		ctx = context.WithValue(ctx, "token_type", tokenType)
	}
	return ctx
}

func assertHTTPError(t *testing.T, err error, status int, reason string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error")
	}
	gotStatus, payload := httperror.Handle(err)
	if gotStatus != status {
		t.Fatalf("expected status %d, got %d with payload %#v", status, gotStatus, payload)
	}
	body, ok := payload.(httperror.Payload)
	if !ok {
		t.Fatalf("unexpected error payload type: %#v", payload)
	}
	if body.Reason != reason {
		t.Fatalf("expected reason %q, got %#v", reason, body)
	}
}
