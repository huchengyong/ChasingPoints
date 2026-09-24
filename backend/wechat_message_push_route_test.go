package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"chasing_points/internal/config"

	"github.com/zeromicro/go-zero/core/logx/logtest"
	resthandler "github.com/zeromicro/go-zero/rest/handler"
)

func TestWechatMessagePushRouteBypassesAccessLogs(t *testing.T) {
	logs := logtest.NewCollector(t)
	const query = "signature=f464b24fc39322e44b38aa78f5edd27bd1441696&timestamp=1714036504&nonce=1514711492&echostr=secret-challenge"

	for _, logger := range []struct {
		name string
		wrap func(http.Handler) http.Handler
	}{
		{name: "brief", wrap: resthandler.LogHandler},
		{name: "detailed", wrap: resthandler.DetailedLogHandler},
	} {
		for _, tc := range []struct {
			name  string
			path  string
			token string
			code  int
		}{
			{name: "valid", path: "/api/wechat/message-push", token: "AAAAA", code: http.StatusOK},
			{name: "trailing slash", path: "/api/wechat/message-push/", token: "AAAAA", code: http.StatusOK},
			{name: "missing token", path: "/api/wechat/message-push", code: http.StatusServiceUnavailable},
		} {
			t.Run(logger.name+"/"+tc.name, func(t *testing.T) {
				logs.Reset()
				cfg := config.Config{}
				cfg.WechatMiniProgram.MessagePushToken = tc.token
				server := &http.Server{
					Handler: logger.wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
						t.Fatal("message push request must not reach go-zero handlers")
					})),
				}

				withWechatMessagePushRoute(cfg)(server)
				rr := httptest.NewRecorder()
				server.Handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, tc.path+"?"+query, nil))

				if rr.Code != tc.code {
					t.Fatalf("status = %d, want %d", rr.Code, tc.code)
				}
				if strings.Contains(logs.String(), "secret-challenge") {
					t.Fatal("access log must not contain echostr")
				}
			})
		}
	}
}

func TestWechatMessagePushRouteRejectsPostWithoutAccessLog(t *testing.T) {
	logs := logtest.NewCollector(t)
	cfg := config.Config{}
	cfg.WechatMiniProgram.MessagePushToken = "AAAAA"
	server := &http.Server{
		Handler: resthandler.DetailedLogHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Fatal("message push POST request must not reach go-zero handlers")
		})),
	}

	withWechatMessagePushRoute(cfg)(server)
	rr := httptest.NewRecorder()
	server.Handler.ServeHTTP(
		rr,
		httptest.NewRequest(http.MethodPost, "/api/wechat/message-push?echostr=secret-challenge", nil),
	)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
	if got := rr.Header().Get("Allow"); got != http.MethodGet {
		t.Fatalf("Allow = %q, want %q", got, http.MethodGet)
	}
	if strings.Contains(logs.String(), "secret-challenge") {
		t.Fatal("access log must not contain echostr")
	}
}

func TestWechatMessagePushRouteLeavesOtherPathsUntouched(t *testing.T) {
	called := false
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusNoContent)
		}),
	}

	withWechatMessagePushRoute(config.Config{})(server)
	rr := httptest.NewRecorder()
	server.Handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/other?echostr=secret-challenge", nil))

	if !called {
		t.Fatal("non-message-push request must reach the original handler")
	}
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}
