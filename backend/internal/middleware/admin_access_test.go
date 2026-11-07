package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// serveAdminAccess runs the middleware over a recording next handler and
// reports the response plus whether the business handler was reached.
func serveAdminAccess(t *testing.T, path string, ctx context.Context) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	reached := false
	handler := NewAdminAccessMiddleware().Handle(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil).WithContext(ctx)
	req.URL.Path = path
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec, reached
}

func TestAdminAccessLetsAdministratorsThrough(t *testing.T) {
	for _, path := range []string{"/api/admin/users/list", "/api/admin/gear/product/save", "/api/admin/event-news/match/create"} {
		rec, reached := serveAdminAccess(t, path, withUserID(-1))
		if !reached || rec.Code != http.StatusOK {
			t.Fatalf("%s: administrator must reach the handler, reached=%v code=%d", path, reached, rec.Code)
		}
	}
}

func TestAdminAccessRejectsRegularUsers(t *testing.T) {
	for _, id := range []int64{1, 3001, 1 << 40} {
		rec, reached := serveAdminAccess(t, "/api/admin/users/update-status", withUserID(id))
		if reached {
			t.Fatalf("user %d must not reach an admin handler", id)
		}
		if rec.Code != http.StatusForbidden {
			t.Fatalf("user %d: expected 403, got %d", id, rec.Code)
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("user %d: parse body: %v", id, err)
		}
		if body["success"] != false || body["code"] != float64(http.StatusForbidden) || body["message"] != "无管理员权限" {
			t.Fatalf("user %d: unexpected body %+v", id, body)
		}
	}
}

func TestAdminAccessRequiresAnIdentity(t *testing.T) {
	cases := map[string]context.Context{
		"anonymous":     context.Background(),
		"zero id":       withUserID(0),
		"unparsable id": context.WithValue(context.Background(), "user_id", "not-a-number"),
	}
	for name, ctx := range cases {
		rec, reached := serveAdminAccess(t, "/api/admin/dashboard/stats", ctx)
		if reached || rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s: expected 401 without reaching the handler, reached=%v code=%d", name, reached, rec.Code)
		}
	}
}

func TestAdminAccessKeepsPublicEntriesAnonymous(t *testing.T) {
	// A stale or foreign user token in the header must not break these entries.
	contexts := map[string]context.Context{
		"anonymous": context.Background(),
		"user":      withUserID(3001),
		"admin":     withUserID(-1),
	}
	for _, path := range []string{"/api/admin/login", "/api/admin/init", "/api/admin/exists"} {
		for name, ctx := range contexts {
			rec, reached := serveAdminAccess(t, path, ctx)
			if !reached || rec.Code != http.StatusOK {
				t.Fatalf("%s as %s: public entry must be served, reached=%v code=%d", path, name, reached, rec.Code)
			}
		}
	}
}

func TestAdminAccessJudgesTheCleanedPath(t *testing.T) {
	// go-zero cleans the path before routing, so these all reach /api/admin/users/list.
	for _, path := range []string{
		"/api//admin/users/list",
		"//api/admin/users/list",
		"/api/admin//users/list",
		"/api/admin/./users/list",
		"/api/x/../admin/users/list",
		"/api/admin/users/list/",
		"/api/admin/users/../users/list",
	} {
		rec, reached := serveAdminAccess(t, path, withUserID(3001))
		if reached || rec.Code != http.StatusForbidden {
			t.Fatalf("%s: user token must be rejected, reached=%v code=%d", path, reached, rec.Code)
		}
	}
}

func TestAdminAccessCleaningDoesNotTurnAdminPathsIntoPublicOnes(t *testing.T) {
	// Cleaning may only widen what counts as an admin path, never what counts as public.
	for _, path := range []string{"/api/admin/login/../users/list", "/api/admin/exists/x", "/api/admin/init2"} {
		rec, reached := serveAdminAccess(t, path, withUserID(3001))
		if reached || rec.Code != http.StatusForbidden {
			t.Fatalf("%s: must stay protected, reached=%v code=%d", path, reached, rec.Code)
		}
	}
	// A path that cleans into a public entry is the public entry.
	rec, reached := serveAdminAccess(t, "/api/admin/users/../login", withUserID(3001))
	if !reached || rec.Code != http.StatusOK {
		t.Fatalf("path cleaning to /api/admin/login must be served as login, reached=%v code=%d", reached, rec.Code)
	}
}

func TestAdminAccessLeavesOtherPathsAlone(t *testing.T) {
	for _, path := range []string{"/api/user/info", "/api/public/rank/configs", "/api/administrator", "/api/admins/list", "/", "/api/user/admin/x"} {
		rec, reached := serveAdminAccess(t, path, withUserID(3001))
		if !reached || rec.Code != http.StatusOK {
			t.Fatalf("%s: non-admin path must not be affected, reached=%v code=%d", path, reached, rec.Code)
		}
	}
}

func TestAdminAccessProtectsTheAdminRoot(t *testing.T) {
	for _, path := range []string{"/api/admin", "/api/admin/"} {
		rec, reached := serveAdminAccess(t, path, withUserID(3001))
		if reached || rec.Code != http.StatusForbidden {
			t.Fatalf("%s: user token must be rejected, reached=%v code=%d", path, reached, rec.Code)
		}
	}
}
