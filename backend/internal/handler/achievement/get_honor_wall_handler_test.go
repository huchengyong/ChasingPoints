package achievement

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg/httperror"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetHonorWallHandlerReturnsSafeEmptyResponseWithoutUserContext(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/achievement/honor-wall", nil)

	GetHonorWallHandler(&svc.ServiceContext{}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"success":false`) || !strings.Contains(recorder.Body.String(), `"season_state":"not_started"`) {
		t.Fatalf("expected safe unsuccessful response with lifecycle state, got %s", recorder.Body.String())
	}
}

func TestGetHonorWallHandlerReturnsForbiddenForNonFriend(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Friend{}, &model.FriendBlacklist{}); err != nil {
		t.Fatalf("prepare honor wall permission schema: %v", err)
	}

	requester := &model.User{Id: 1001, Nickname: "查看者", Status: 1}
	target := &model.User{Id: 2002, Nickname: "目标用户", Status: 1}
	if err := db.Create(&[]model.User{*requester, *target}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	svcCtx := &svc.ServiceContext{
		UserModel:   model.NewUserModel(db),
		FriendModel: model.NewFriendModel(db),
	}
	httperror.Configure()

	req := httptest.NewRequest(http.MethodGet, "/api/achievement/honor-wall?user_id=2002", nil)
	req = req.WithContext(context.WithValue(req.Context(), "user_id", int64(1001)))
	rec := httptest.NewRecorder()
	GetHonorWallHandler(svcCtx).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected HTTP 403, got %d body=%s", rec.Code, rec.Body.String())
	}
	var payload httperror.Payload
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode forbidden payload: %v", err)
	}
	if payload.Success || payload.Reason != "HONOR_WALL_FORBIDDEN" || payload.Message == "" {
		t.Fatalf("unexpected forbidden payload: %#v", payload)
	}
}
