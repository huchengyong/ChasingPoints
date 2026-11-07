package season

import (
	"net/http/httptest"
	"strings"
	"testing"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetCurrentSeasonHandlerReturnsLifecycleState(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Season{}); err != nil {
		t.Fatalf("migrate seasons: %v", err)
	}
	svcCtx := &svc.ServiceContext{
		Config:      config.Config{},
		DB:          db,
		SeasonModel: model.NewSeasonModel(db),
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/api/season/current", nil)

	GetCurrentSeasonHandler(svcCtx).ServeHTTP(recorder, request)

	if recorder.Code != 200 {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"success":true`) || !strings.Contains(recorder.Body.String(), `"season_state":"not_started"`) || !strings.Contains(recorder.Body.String(), `"season":null`) {
		t.Fatalf("unexpected lifecycle payload: %s", recorder.Body.String())
	}
}
