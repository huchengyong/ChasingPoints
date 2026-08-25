package match

import (
	"context"
	"net/http"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg"
	"chasing_points/internal/pkg/httperror"
	"chasing_points/internal/pkg/wsticket"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestIssueMatchWSTicketSignsPrivateParticipant(t *testing.T) {
	svcCtx := newMatchWSTicketTestSvc(t)
	seedMatchWSTicketUsers(t, svcCtx, 21, 22, 23)
	opponentID := int64(22)
	refereeID := int64(23)
	seedMatchWSTicketMatch(t, svcCtx, &model.Match{
		Id:            31,
		UserId:        21,
		OpponentId:    &opponentID,
		RefereeUserId: &refereeID,
		MatchMode:     model.MatchModePractice,
		Visibility:    model.MatchVisibilityPrivate,
		Status:        1,
	})
	store := &fakeMatchWSTicketStore{ticket: "match-ticket", ttl: 28 * time.Second}
	svcCtx.WSTicketStore = store

	resp, err := NewIssueMatchWSTicketLogic(matchWSTicketCtx(22, pkg.AccessTokenType), svcCtx).
		IssueMatchWSTicket(&types.MatchWSTicketReq{MatchId: 31})
	if err != nil {
		t.Fatalf("issue match ws ticket: %v", err)
	}
	if !resp.Success || resp.Ticket != "match-ticket" || resp.Scope != wsticket.ScopeMatch || resp.ExpiresInSeconds != 28 {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if len(store.issued) != 1 || store.issued[0].UserID != 22 || store.issued[0].Scope != wsticket.ScopeMatch || store.issued[0].MatchID != 31 {
		t.Fatalf("unexpected issued claims: %+v", store.issued)
	}
}

func TestIssueMatchWSTicketRejectsPrivateNonParticipant(t *testing.T) {
	svcCtx := newMatchWSTicketTestSvc(t)
	seedMatchWSTicketUsers(t, svcCtx, 31, 32, 33)
	opponentID := int64(32)
	seedMatchWSTicketMatch(t, svcCtx, &model.Match{
		Id:         41,
		UserId:     31,
		OpponentId: &opponentID,
		MatchMode:  model.MatchModePractice,
		Visibility: model.MatchVisibilityPrivate,
		Status:     1,
	})
	store := &fakeMatchWSTicketStore{ticket: "unused", ttl: wsticket.DefaultTTL}
	svcCtx.WSTicketStore = store

	_, err := NewIssueMatchWSTicketLogic(matchWSTicketCtx(33, pkg.AccessTokenType), svcCtx).
		IssueMatchWSTicket(&types.MatchWSTicketReq{MatchId: 41})
	assertMatchHTTPError(t, err, http.StatusForbidden, "MATCH_WS_FORBIDDEN")
	if len(store.issued) != 0 {
		t.Fatalf("forbidden user must not receive ticket: %+v", store.issued)
	}
}

func TestIssueMatchWSTicketAllowsPublicReadTicketForLoggedInViewer(t *testing.T) {
	svcCtx := newMatchWSTicketTestSvc(t)
	seedMatchWSTicketUsers(t, svcCtx, 51, 52, 53)
	opponentID := int64(52)
	seedMatchWSTicketMatch(t, svcCtx, &model.Match{
		Id:         61,
		UserId:     51,
		OpponentId: &opponentID,
		Visibility: model.MatchVisibilityPublic,
		Status:     1,
	})
	store := &fakeMatchWSTicketStore{ticket: "public-match-ticket", ttl: wsticket.DefaultTTL}
	svcCtx.WSTicketStore = store

	resp, err := NewIssueMatchWSTicketLogic(matchWSTicketCtx(53, pkg.AccessTokenType), svcCtx).
		IssueMatchWSTicket(&types.MatchWSTicketReq{MatchId: 61})
	if err != nil {
		t.Fatalf("issue public match ws ticket: %v", err)
	}
	if !resp.Success || resp.Scope != wsticket.ScopeMatch {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if len(store.issued) != 1 || store.issued[0].UserID != 53 || store.issued[0].MatchID != 61 {
		t.Fatalf("unexpected issued claims: %+v", store.issued)
	}
}

type fakeMatchWSTicketStore struct {
	ticket string
	ttl    time.Duration
	issued []wsticket.Claims
}

func (s *fakeMatchWSTicketStore) Issue(ctx context.Context, claims wsticket.Claims, ttl time.Duration) (string, time.Duration, error) {
	if s.ticket == "" {
		s.ticket = "fake-match-ticket"
	}
	if s.ttl <= 0 {
		s.ttl = wsticket.DefaultTTL
	}
	s.issued = append(s.issued, claims)
	return s.ticket, s.ttl, nil
}

func (s *fakeMatchWSTicketStore) Consume(ctx context.Context, ticket string, expectedScope string) (*wsticket.Claims, error) {
	return nil, wsticket.ErrTicketNotFound
}

func newMatchWSTicketTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}); err != nil {
		t.Fatalf("prepare match ws ticket schema: %v", err)
	}
	return &svc.ServiceContext{
		DB:         db,
		UserModel:  model.NewUserModel(db),
		MatchModel: model.NewMatchModel(db),
	}
}

func seedMatchWSTicketUsers(t *testing.T, svcCtx *svc.ServiceContext, ids ...int64) {
	t.Helper()
	for _, id := range ids {
		if err := svcCtx.UserModel.Create(&model.User{Id: id, Nickname: "WS用户", Status: 1}); err != nil {
			t.Fatalf("create user %d: %v", id, err)
		}
	}
}

func seedMatchWSTicketMatch(t *testing.T, svcCtx *svc.ServiceContext, match *model.Match) {
	t.Helper()
	if err := svcCtx.MatchModel.Create(match); err != nil {
		t.Fatalf("create match: %v", err)
	}
}

func matchWSTicketCtx(userID int64, tokenType string) context.Context {
	ctx := context.WithValue(context.Background(), "user_id", userID)
	if tokenType != "" {
		ctx = context.WithValue(ctx, "token_type", tokenType)
	}
	return ctx
}

func assertMatchHTTPError(t *testing.T, err error, status int, reason string) {
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
