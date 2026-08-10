package achievement

import (
	"net/http/httptest"
	"strings"
	"testing"

	"chasing_points/internal/svc"
)

func TestGetHonorWallHandlerReturnsSafeEmptyResponseWithoutUserContext(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/api/achievement/honor-wall", nil)

	GetHonorWallHandler(&svc.ServiceContext{}).ServeHTTP(recorder, request)

	if recorder.Code != 200 {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"success":false`) || !strings.Contains(recorder.Body.String(), `"season_state":"not_started"`) {
		t.Fatalf("expected safe unsuccessful response with lifecycle state, got %s", recorder.Body.String())
	}
}
