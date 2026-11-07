package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"
	"chasing_points/internal/utils"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAdminHandlerTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	miniRedis := miniredis.RunT(t)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	schema := []string{
		`CREATE TABLE admins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			nickname TEXT NOT NULL,
			avatar TEXT DEFAULT '',
			role TEXT NOT NULL,
			status INTEGER NOT NULL DEFAULT 1,
			last_login_at DATETIME NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE admin_login_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			admin_id INTEGER NOT NULL,
			ip TEXT DEFAULT '',
			user_agent TEXT DEFAULT '',
			login_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
	}
	for _, stmt := range schema {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("create test schema: %v", err)
		}
	}
	if err := testsupport.PrepareFavoriteVenueRewardSchema(db); err != nil {
		t.Fatalf("prepare favorite venue reward schema: %v", err)
	}

	var cfg config.Config
	cfg.Auth.AccessSecret = "test-secret"
	cfg.Auth.AccessExpire = 3600
	cfg.Admin.SetupToken = "setup-secret"

	return &svc.ServiceContext{
		Config:             cfg,
		DB:                 db,
		Redis:              redis.NewClient(&redis.Options{Addr: miniRedis.Addr()}),
		AdminModel:         model.NewAdminModel(db),
		AdminLoginLogModel: model.NewAdminLoginLogModel(db),
		FavoriteVenueRewardConfigModel: model.NewFavoriteVenueRewardConfigModel(db),
	}
}

func createHandlerAdmin(t *testing.T, svcCtx *svc.ServiceContext, email, password string) *model.Admin {
	t.Helper()

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	admin := &model.Admin{
		Email:    email,
		Password: hashedPassword,
		Nickname: "管理员",
		Role:     "super_admin",
		Status:   1,
	}
	if err := svcCtx.AdminModel.Create(context.Background(), admin); err != nil {
		t.Fatalf("create admin: %v", err)
	}

	return admin
}

func TestAdminInitHandlerAcceptsCamelCaseSetupToken(t *testing.T) {
	svcCtx := newAdminHandlerTestSvc(t)

	body := `{"email":"1437136500@qq.com","password":"Hcy@1437136500","setupToken":"setup-secret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/init", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	AdminInitHandler(svcCtx).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["success"] != true {
		t.Fatalf("expected success response, got %v", resp)
	}
}

func TestAdminInitHandlerReturnsJSONOnInvalidJSON(t *testing.T) {
	svcCtx := newAdminHandlerTestSvc(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/init", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	AdminInitHandler(svcCtx).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON error response, got %q: %v", rr.Body.String(), err)
	}
	if resp["success"] != false {
		t.Fatalf("expected failure response, got %v", resp)
	}
}

func TestAdminUpdateVenueRewardConfigHandlerAllowsMissingNewUserWindowDays(t *testing.T) {
	svcCtx := newAdminHandlerTestSvc(t)

	body := `{"enabled":false,"popup_enabled":false,"reward_days":30,"welcome_reward_enabled":false,"welcome_reward_days":7,"start_at":"","end_at":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/venue/reward-config", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	AdminUpdateVenueRewardConfigHandler(svcCtx).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["success"] != true {
		t.Fatalf("expected success response, got %v", resp)
	}
}

func TestAdminChangePasswordHandlerAcceptsCamelCaseFields(t *testing.T) {
	svcCtx := newAdminHandlerTestSvc(t)
	admin := createHandlerAdmin(t, svcCtx, "admin@example.com", "Password123!")

	body := `{"oldPassword":"Password123!","newPassword":"NewPassword123!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/user/change-password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), "user_id", -int64(admin.Id)))
	rr := httptest.NewRecorder()

	AdminChangePasswordHandler(svcCtx).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["success"] != true {
		t.Fatalf("expected success response, got %v", resp)
	}
}
