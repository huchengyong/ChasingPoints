package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"chasing_points/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newActiveUserSessionTestSvc(t *testing.T) (*model.UserModel, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate users: %v", err)
	}
	return model.NewUserModel(db), db
}

func seedActiveUserSessionUser(t *testing.T, db *gorm.DB, id int64, status int) {
	t.Helper()
	if err := db.Create(&model.User{Id: id, Nickname: "用户", Status: status}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	// GORM omits zero-valued fields with a column default (Status = 0 would be
	// written as the default 1), so force the disabled status with an UPDATE.
	if err := db.Model(&model.User{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		t.Fatalf("update user status: %v", err)
	}
}

func withUserID(id int64) context.Context {
	return context.WithValue(context.Background(), "user_id", id)
}

func runActiveUserSessionMiddleware(t *testing.T, userModel *model.UserModel, ctx context.Context) (int, map[string]any) {
	t.Helper()
	mw := NewActiveUserSessionMiddleware(userModel)
	handler := mw.Handle(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler(rec, req)

	body := map[string]any{}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec.Code, body
}

func TestActiveUserSessionMiddlewareAllowsValidUser(t *testing.T) {
	userModel, db := newActiveUserSessionTestSvc(t)
	seedActiveUserSessionUser(t, db, 1001, 1)

	code, _ := runActiveUserSessionMiddleware(t, userModel, withUserID(1001))
	if code != http.StatusOK {
		t.Fatalf("expected 200 for valid user, got %d", code)
	}
}

func TestActiveUserSessionMiddlewareRejectsMissingUser(t *testing.T) {
	userModel, _ := newActiveUserSessionTestSvc(t)

	code, body := runActiveUserSessionMiddleware(t, userModel, withUserID(9999))
	if code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing user, got %d", code)
	}
	if body["reason"] != SessionInvalidReason || body["success"] != false {
		t.Fatalf("unexpected invalid payload: %+v", body)
	}
}

func TestActiveUserSessionMiddlewareRejectsDisabledUser(t *testing.T) {
	userModel, db := newActiveUserSessionTestSvc(t)
	seedActiveUserSessionUser(t, db, 1002, 0)

	code, body := runActiveUserSessionMiddleware(t, userModel, withUserID(1002))
	if code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for disabled user, got %d", code)
	}
	if body["reason"] != SessionInvalidReason {
		t.Fatalf("expected SESSION_INVALID reason, got %+v", body)
	}
}

func TestActiveUserSessionMiddlewareAllowsAnonymousRequest(t *testing.T) {
	userModel, _ := newActiveUserSessionTestSvc(t)

	code, _ := runActiveUserSessionMiddleware(t, userModel, context.Background())
	if code != http.StatusOK {
		t.Fatalf("expected 200 for anonymous request, got %d", code)
	}
}

func TestActiveUserSessionMiddlewareAllowsAdminIdentity(t *testing.T) {
	userModel, _ := newActiveUserSessionTestSvc(t)

	code, _ := runActiveUserSessionMiddleware(t, userModel, withUserID(-7))
	if code != http.StatusOK {
		t.Fatalf("expected 200 for admin identity, got %d", code)
	}
}

func TestActiveUserSessionMiddlewareReturnsServerErrorOnQueryFailure(t *testing.T) {
	userModel, db := newActiveUserSessionTestSvc(t)
	seedActiveUserSessionUser(t, db, 1003, 1)

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	code, _ := runActiveUserSessionMiddleware(t, userModel, withUserID(1003))
	if code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on query failure, got %d", code)
	}
}
