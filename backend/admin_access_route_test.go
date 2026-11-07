package main

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/handler"
	"chasing_points/internal/model"
	"chasing_points/internal/pkg"
	"chasing_points/internal/svc"

	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/rest"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const adminAccessTestSecret = "admin-access-route-secret"

// publicAdminRoutes are the only /api/admin routes that may be served without an
// administrator identity. They are not requested here: their handlers need the
// database, and what matters is that nothing else is on this list.
var publicAdminRoutes = map[string]bool{
	"POST /api/admin/login": true,
	"POST /api/admin/init":  true,
	"GET /api/admin/exists": true,
}

// startAdminAccessServer runs the real go-zero server with every generated route
// and the same global middlewares main() installs, backed by an in-memory user table.
func startAdminAccessServer(t *testing.T) (*rest.Server, string) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve a port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	var cfg config.Config
	cfg.RestConf = rest.RestConf{ServiceConf: service.ServiceConf{Name: "admin-access-test"}, Host: "127.0.0.1", Port: port}
	cfg.Auth.AccessSecret = adminAccessTestSecret

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate users: %v", err)
	}
	if err := db.Create(&model.User{Id: 3001, Nickname: "用户", Status: 1}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	svcCtx := &svc.ServiceContext{Config: cfg, UserModel: model.NewUserModel(db)}

	server := newAPIServer(cfg)
	handler.RegisterHandlers(server, svcCtx)
	registerGlobalMiddlewares(server, cfg, svcCtx)
	go server.Start()
	t.Cleanup(server.Stop)

	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.Now().Add(5 * time.Second)
	for {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return server, base
		}
		if time.Now().After(deadline) {
			t.Fatalf("server did not start: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func adminAccessStatus(t *testing.T, method, url, token string) int {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

// TestEveryAdminRouteRejectsRegularUsers is the regression guard for the admin
// access boundary: before it existed, 20 of 39 JWT-protected admin routes could
// be called with any logged-in user's token. It walks the routes actually
// registered, so a new admin route is covered the moment it is added.
func TestEveryAdminRouteRejectsRegularUsers(t *testing.T) {
	server, base := startAdminAccessServer(t)
	userToken, err := pkg.GenerateTypedToken(3001, adminAccessTestSecret, 600, "access")
	if err != nil {
		t.Fatalf("generate user token: %v", err)
	}

	var protected []rest.Route
	registeredPublic := map[string]bool{}
	for _, route := range server.Routes() {
		if route.Path != "/api/admin" && !strings.HasPrefix(route.Path, "/api/admin/") {
			continue
		}
		key := route.Method + " " + route.Path
		if publicAdminRoutes[key] {
			registeredPublic[key] = true
			continue
		}
		if strings.Contains(route.Path, ":") {
			t.Fatalf("%s: path parameters need their own cases in this test", key)
		}
		protected = append(protected, route)
	}

	for key := range publicAdminRoutes {
		if !registeredPublic[key] {
			t.Errorf("public admin route %s is not registered; the allow list is stale", key)
		}
	}
	// Guards against the walk silently covering nothing (39 JWT-protected admin routes at the time of writing).
	if len(protected) < 39 {
		t.Fatalf("only %d protected admin routes found; the route walk is broken", len(protected))
	}

	for _, route := range protected {
		url := base + route.Path
		if got := adminAccessStatus(t, route.Method, url, userToken); got != http.StatusForbidden {
			t.Errorf("%s %s with a regular user token = %d, want 403", route.Method, route.Path, got)
		}
		if got := adminAccessStatus(t, route.Method, url, ""); got != http.StatusUnauthorized {
			t.Errorf("%s %s without a token = %d, want 401", route.Method, route.Path, got)
		}

		// The router cleans the path before matching, so these spellings reach the same handler.
		suffix := strings.TrimPrefix(route.Path, "/api/admin")
		for _, variant := range []string{"/api//admin" + suffix, "/api/./admin" + suffix, "/api/x/../admin" + suffix} {
			if got := adminAccessStatus(t, route.Method, base+variant, userToken); got != http.StatusForbidden {
				t.Errorf("%s %s with a regular user token = %d, want 403", route.Method, variant, got)
			}
		}
	}
}

func TestAdminAccessLeavesUserRoutesAlone(t *testing.T) {
	_, base := startAdminAccessServer(t)
	userToken, err := pkg.GenerateTypedToken(3001, adminAccessTestSecret, 600, "access")
	if err != nil {
		t.Fatalf("generate user token: %v", err)
	}
	if got := adminAccessStatus(t, http.MethodGet, base+"/api/user/info", userToken); got != http.StatusOK {
		t.Fatalf("/api/user/info with a regular user token = %d, want 200", got)
	}
}
