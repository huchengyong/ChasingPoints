package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	pkg "chasing_points/internal/pkg"

	"github.com/zeromicro/go-zero/rest/handler"
)

// TestActiveUserSessionMiddlewareRunsAfterJWT verifies the real chain order:
// go-zero's JWT Authorize runs first and only then the active-user session
// middleware executes its user lookup, so a signed token for a deleted user
// is rejected before the business handler runs.
func TestActiveUserSessionMiddlewareRunsAfterJWT(t *testing.T) {
	const secret = "session-integration-secret"
	userModel, db := newActiveUserSessionTestSvc(t)
	seedActiveUserSessionUser(t, db, 2001, 1)

	var businessRan bool
	business := func(w http.ResponseWriter, r *http.Request) {
		businessRan = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}

	auth := handler.Authorize(secret)
	handled := auth(NewActiveUserSessionMiddleware(userModel).Handle(business))

	validToken, err := pkg.GenerateTypedToken(2001, secret, 600, "access")
	if err != nil {
		t.Fatalf("generate valid token: %v", err)
	}
	deletedUserToken, err := pkg.GenerateTypedToken(9999, secret, 600, "access")
	if err != nil {
		t.Fatalf("generate deleted user token: %v", err)
	}
	refreshToken, err := pkg.GenerateTypedToken(2001, secret, 600, "refresh")
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}

	t.Run("valid user reaches handler", func(t *testing.T) {
		businessRan = false
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		rec := httptest.NewRecorder()
		handled.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK || !businessRan {
			t.Fatalf("expected handler to run with 200, got code=%d ran=%v", rec.Code, businessRan)
		}
	})

	t.Run("deleted user blocked before handler", func(t *testing.T) {
		businessRan = false
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+deletedUserToken)
		rec := httptest.NewRecorder()
		handled.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
		if businessRan {
			t.Fatal("business handler must not run for a deleted user session")
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("parse body: %v", err)
		}
		if body["reason"] != SessionInvalidReason {
			t.Fatalf("expected SESSION_INVALID reason, got %+v", body)
		}
	})

	t.Run("refresh token cannot access business route", func(t *testing.T) {
		businessRan = false
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+refreshToken)
		rec := httptest.NewRecorder()
		handled.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for refresh token, got %d", rec.Code)
		}
		if businessRan {
			t.Fatal("business handler must not run with a refresh token")
		}
	})

	t.Run("missing token rejected by jwt layer", func(t *testing.T) {
		businessRan = false
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handled.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 from jwt layer, got %d", rec.Code)
		}
		if businessRan {
			t.Fatal("business handler must not run without a token")
		}
	})
}
