package logic

import (
	"context"
	"testing"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAdminTestSvc(t *testing.T) (*svc.ServiceContext, *miniredis.Miniredis) {
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
	}, miniRedis
}

func createAdmin(t *testing.T, svcCtx *svc.ServiceContext, email, password string) *model.Admin {
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

func TestAdminInitRequiresSetupTokenAndNormalizesEmail(t *testing.T) {
	svcCtx, _ := newAdminTestSvc(t)
	logic := NewAdminInitLogic(context.Background(), svcCtx)

	resp, err := logic.AdminInit(&types.AdminInitReq{
		Email:      "Admin@Example.com",
		Password:   "Password123!",
		SetupToken: "wrong-token",
	})
	if err != nil {
		t.Fatalf("admin init with wrong token returned error: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected wrong setup token to be rejected")
	}
	if resp.Code != 403 {
		t.Fatalf("expected 403 for wrong setup token, got %d", resp.Code)
	}

	resp, err = logic.AdminInit(&types.AdminInitReq{
		Email:      "Admin@Example.com",
		Password:   "Password123!",
		SetupToken: "setup-secret",
	})
	if err != nil {
		t.Fatalf("admin init returned error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected init success, got %#v", resp)
	}

	admin, err := svcCtx.AdminModel.FindByEmail(context.Background(), "admin@example.com")
	if err != nil {
		t.Fatalf("find normalized admin email: %v", err)
	}
	if admin.Email != "admin@example.com" {
		t.Fatalf("expected normalized email, got %q", admin.Email)
	}
}

func TestAdminLoginUsesNormalizedEmailForLookupAndRateLimit(t *testing.T) {
	svcCtx, miniRedis := newAdminTestSvc(t)
	createAdmin(t, svcCtx, "admin@example.com", "Password123!")

	logic := NewAdminLoginLogic(context.Background(), svcCtx)

	resp, err := logic.AdminLogin(&types.AdminLoginReq{
		Email:    "ADMIN@EXAMPLE.COM",
		Password: "wrong-password",
	})
	if err != nil {
		t.Fatalf("admin login returned error: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected wrong password to fail")
	}

	if !miniRedis.Exists("admin:login:fail:admin@example.com") {
		t.Fatalf("expected normalized rate-limit key to exist")
	}
	if miniRedis.Exists("admin:login:fail:ADMIN@EXAMPLE.COM") {
		t.Fatalf("expected raw email rate-limit key to be absent")
	}

	resp, err = logic.AdminLogin(&types.AdminLoginReq{
		Email:    "ADMIN@EXAMPLE.COM",
		Password: "Password123!",
	})
	if err != nil {
		t.Fatalf("admin login with normalized email returned error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected normalized email login to succeed, got %#v", resp)
	}
}

func TestAdminChangePasswordRequiresCorrectOldPassword(t *testing.T) {
	svcCtx, _ := newAdminTestSvc(t)
	admin := createAdmin(t, svcCtx, "admin@example.com", "Password123!")

	ctx := context.WithValue(context.Background(), "user_id", -int64(admin.Id))
	logic := NewAdminChangePasswordLogic(ctx, svcCtx)

	resp, err := logic.AdminChangePassword(&types.AdminChangePasswordReq{
		OldPassword: "bad-password",
		NewPassword: "NewPassword123!",
	})
	if err != nil {
		t.Fatalf("change password with wrong old password returned error: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected wrong old password to fail")
	}
	if resp.Code != 400 {
		t.Fatalf("expected 400 for wrong old password, got %d", resp.Code)
	}

	resp, err = logic.AdminChangePassword(&types.AdminChangePasswordReq{
		OldPassword: "Password123!",
		NewPassword: "NewPassword123!",
	})
	if err != nil {
		t.Fatalf("change password returned error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected password change success, got %#v", resp)
	}

	updatedAdmin, err := svcCtx.AdminModel.FindById(context.Background(), admin.Id)
	if err != nil {
		t.Fatalf("find updated admin: %v", err)
	}
	if !utils.CheckPasswordHash("NewPassword123!", updatedAdmin.Password) {
		t.Fatalf("expected new password hash to be persisted")
	}
}
