package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"chasing_points/internal/handler/user"
	"chasing_points/internal/svc"
	pkg "chasing_points/internal/pkg"

	"github.com/zeromicro/go-zero/rest/handler"
)

// TestUserInfoRouteRejectsDeletedUserSession is a route-level regression test.
// Previously a signed JWT for a deleted user reached GetUserInfoLogic which
// returned HTTP 200 with an empty success:false body; the active-user session
// middleware must now turn that into a standard 401 SESSION_INVALID while
// keeping the valid-user response unchanged.
func TestUserInfoRouteRejectsDeletedUserSession(t *testing.T) {
	const secret = "user-info-route-secret"
	userModel, db := newActiveUserSessionTestSvc(t)
	seedActiveUserSessionUser(t, db, 3001, 1)

	svcCtx := &svc.ServiceContext{UserModel: userModel}
	business := user.GetUserInfoHandler(svcCtx)
	handled := handler.Authorize(secret)(NewActiveUserSessionMiddleware(userModel).Handle(business))

	validToken, err := pkg.GenerateTypedToken(3001, secret, 600, "access")
	if err != nil {
		t.Fatalf("generate valid token: %v", err)
	}
	deletedUserToken, err := pkg.GenerateTypedToken(9999, secret, 600, "access")
	if err != nil {
		t.Fatalf("generate deleted user token: %v", err)
	}

	t.Run("valid user still receives success payload", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/user/info", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		rec := httptest.NewRecorder()
		handled.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 for valid user, got %d", rec.Code)
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("parse body: %v", err)
		}
		if body["success"] != true {
			t.Fatalf("expected success true, got %+v", body)
		}
		if body["user_info"] == nil {
			t.Fatalf("expected user_info payload, got %+v", body)
		}
	})

	t.Run("deleted user gets 401 instead of 200 empty failure", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/user/info", nil)
		req.Header.Set("Authorization", "Bearer "+deletedUserToken)
		rec := httptest.NewRecorder()
		handled.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for deleted user, got %d", rec.Code)
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("parse body: %v", err)
		}
		if body["reason"] != SessionInvalidReason {
			t.Fatalf("expected SESSION_INVALID, got %+v", body)
		}
		if body["user_info"] != nil {
			t.Fatalf("deleted user must not receive user_info, got %+v", body)
		}
	})
}
