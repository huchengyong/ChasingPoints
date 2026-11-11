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
		"/api/social/public",
	}
	mux := http.NewServeMux()
	for _, path := range []string{
		"/api/public/rank/leaderboard",
		"/api/public/rank/leaderboard-summary",
		"/api/public/matches",
		"/api/social/public",
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

// Other /api/social routes keep their own JWT handling; the optional middleware
// must see them as a plain passthrough.
func TestOptionalPublicJWTMiddlewareLeavesOtherSocialRoutesUntouched(t *testing.T) {
	const secret = "optional-public-jwt-test-secret"
	handler := NewOptionalPublicJWTMiddleware(secret).Handle(func(w http.ResponseWriter, r *http.Request) {
		userID, err := utils.GetOptionalUserIDFromCtx(r.Context())
		if err != nil {
			t.Fatalf("read optional viewer: %v", err)
		}
		_, _ = w.Write([]byte(strconv.FormatInt(userID, 10)))
	})

	token, err := pkg.GenerateTypedToken(42, secret, 60, pkg.AccessTokenType)
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}
	for _, authorization := range []string{"", "Bearer " + token, "Bearer invalid-token"} {
		request := httptest.NewRequest(http.MethodGet, "/api/social/feed", nil)
		if authorization != "" {
			request.Header.Set("Authorization", authorization)
		}
		response := httptest.NewRecorder()
		handler(response, request)
		if response.Code != http.StatusOK || response.Body.String() != "0" {
			t.Fatalf("unaffected social route with %q = status %d body %q", authorization, response.Code, response.Body.String())
		}
	}
}
