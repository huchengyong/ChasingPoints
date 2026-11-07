package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"chasing_points/internal/pkg"
	"chasing_points/internal/utils"
)

func TestOptionalPublicJWTMiddlewarePreservesPublicRoutesAndAuthenticatedViewer(t *testing.T) {
	const secret = "optional-public-jwt-test-secret"
	paths := []string{
		"/api/public/rank/leaderboard",
		"/api/public/rank/leaderboard-summary",
		"/api/public/matches?scope=friends",
	}
	mux := http.NewServeMux()
	for _, path := range []string{
		"/api/public/rank/leaderboard",
		"/api/public/rank/leaderboard-summary",
		"/api/public/matches",
	} {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			userID, err := utils.GetOptionalUserIDFromCtx(r.Context())
			if err != nil {
				t.Fatalf("read optional viewer: %v", err)
			}
			_, _ = w.Write([]byte(strconv.FormatInt(userID, 10)))
		})
	}
	handler := NewOptionalPublicJWTMiddleware(secret).Handle(mux.ServeHTTP)

	for _, path := range paths {
		response := httptest.NewRecorder()
		handler(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK || response.Body.String() != "0" {
			t.Fatalf("anonymous public request %s = status %d body %q", path, response.Code, response.Body.String())
		}
	}

	token, err := pkg.GenerateTypedToken(42, secret, 60, pkg.AccessTokenType)
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}
	for _, path := range paths {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		handler(response, request)
		if response.Code != http.StatusOK || response.Body.String() != "42" {
			t.Fatalf("authenticated public request %s = status %d body %q", path, response.Code, response.Body.String())
		}
	}

	for _, path := range paths {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Header.Set("Authorization", "Bearer invalid-token")
		response := httptest.NewRecorder()
		handler(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("invalid token must not gain an anonymous or viewer context for %s: status %d", path, response.Code)
		}
	}
}
