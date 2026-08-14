package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	"chasing_points/internal/pkg"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRequestObservabilityMiddlewareRecordsNormalizedCompletion(t *testing.T) {
	var completion RequestCompletion
	handler := NewRequestObservabilityMiddleware(func(_ context.Context, value RequestCompletion) {
		completion = value
	})(func(w http.ResponseWriter, r *http.Request) {
		if got := observability.RequestIDFromContext(r.Context()); got != "client-request-1" {
			t.Fatalf("request id in context = %q", got)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	})

	request := httptest.NewRequest(http.MethodGet, "/api/match/42/6ba7b810-9dad-11d1-80b4-00c04fd430c8?token=secret", nil)
	request.Header.Set(requestIDHeader, "client-request-1")
	response := httptest.NewRecorder()
	handler(response, request)

	if got := response.Header().Get(requestIDHeader); got != "client-request-1" {
		t.Fatalf("response request id = %q", got)
	}
	if completion.Method != http.MethodGet || completion.Route != "/api/match/:id/:id" {
		t.Fatalf("unexpected completion route: %+v", completion)
	}
	if completion.Status != http.StatusCreated || completion.ResponseBytes != 2 {
		t.Fatalf("unexpected completion response: %+v", completion)
	}
	if completion.Duration < 0 {
		t.Fatalf("duration must not be negative: %v", completion.Duration)
	}
}

func TestRequestObservabilityMiddlewareRecordsRequestCacheOutcomes(t *testing.T) {
	var completion RequestCompletion
	handler := NewRequestObservabilityMiddleware(func(_ context.Context, value RequestCompletion) {
		completion = value
	})(func(w http.ResponseWriter, r *http.Request) {
		var cache observability.CacheMetrics
		cache.Hit(r.Context())
		cache.Miss(r.Context())
		cache.DecodeError(r.Context())
		cache.RedisError(r.Context())
		cache.WriteError(r.Context())
		cache.Fallback(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/public/rank/leaderboard", nil))

	if completion.CacheHits != 1 || completion.CacheMisses != 1 || completion.CacheDecodeErrors != 1 ||
		completion.CacheRedisErrors != 1 || completion.CacheWriteErrors != 1 || completion.CacheFallbacks != 1 {
		t.Fatalf("unexpected request cache metrics: %+v", completion)
	}
}

func TestRequestObservabilityMiddlewareRecordsCompletionWhenHandlerPanics(t *testing.T) {
	var completion RequestCompletion
	handler := NewRequestObservabilityMiddleware(func(_ context.Context, value RequestCompletion) {
		completion = value
	})(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})

	func() {
		defer func() { _ = recover() }()
		handler(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/public/rank/leaderboard", nil))
	}()
	if completion.Status != http.StatusInternalServerError || completion.RequestID == "" {
		t.Fatalf("panic path must emit one 500 completion: %+v", completion)
	}
}

func TestRequestObservabilityMiddlewareRecordsAuthAndBusinessSQLForHTTPChain(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate user: %v", err)
	}
	if err := db.Create(&model.User{Id: 7, Nickname: "viewer", Status: 1}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	baseContext := &svc.ServiceContext{DB: db, UserModel: model.NewUserModel(db)}
	var completion RequestCompletion
	business := func(w http.ResponseWriter, r *http.Request) {
		user, findErr := baseContext.WithContext(r.Context()).UserModel.FindById(7)
		if findErr != nil || user == nil {
			t.Fatalf("business user query: user=%+v err=%v", user, findErr)
		}
		w.WriteHeader(http.StatusOK)
	}
	chain := NewRequestObservabilityMiddleware(func(_ context.Context, value RequestCompletion) {
		completion = value
	})(NewOptionalPublicJWTMiddleware("test-secret").Handle(
		NewActiveUserSessionMiddleware(model.NewUserModel(db)).Handle(business),
	))
	token, err := pkg.GenerateTypedToken(7, "test-secret", 60, pkg.AccessTokenType)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/public/rank/leaderboard", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	chain(response, request)
	if response.Code != http.StatusOK || completion.SQLCount < 2 {
		t.Fatalf("HTTP chain must count authentication and business queries: response=%d completion=%+v", response.Code, completion)
	}
}

func TestRequestObservabilityMiddlewareRejectsUnsafeRequestID(t *testing.T) {
	var completion RequestCompletion
	handler := NewRequestObservabilityMiddleware(func(_ context.Context, value RequestCompletion) {
		completion = value
	})(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	request := httptest.NewRequest(http.MethodGet, "/api/user/info", nil)
	request.Header.Set(requestIDHeader, "unsafe request id")
	response := httptest.NewRecorder()
	handler(response, request)

	if completion.RequestID == "unsafe request id" || !isSafeRequestID(completion.RequestID) {
		t.Fatalf("unsafe request id was accepted: %q", completion.RequestID)
	}
	if got := response.Header().Get(requestIDHeader); got != completion.RequestID {
		t.Fatalf("response request id = %q, want %q", got, completion.RequestID)
	}
}

func TestNormalizeRoute(t *testing.T) {
	cases := map[string]string{
		"":                       "/",
		"/":                      "/",
		"/api/notification/123":  "/api/notification/:id",
		"/api/public/event-news": "/api/public/event-news",
		"/api/match/abc-123":     "/api/match/abc-123",
		"/api/match/6ba7b810-9dad-11d1-80b4-00c04fd430c8": "/api/match/:id",
	}
	for input, want := range cases {
		if got := normalizeRoute(input); got != want {
			t.Errorf("normalizeRoute(%q) = %q, want %q", input, got, want)
		}
	}
}
