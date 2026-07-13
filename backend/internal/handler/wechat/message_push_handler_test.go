package wechat

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMessagePushHandlerReturnsChallengeForValidWechatSignature(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/wechat/message-push?signature=f464b24fc39322e44b38aa78f5edd27bd1441696&timestamp=1714036504&nonce=1514711492&echostr=4375120948345356249",
		nil,
	)
	rr := httptest.NewRecorder()

	MessagePushHandler("AAAAA").ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want %q", got, "text/plain; charset=utf-8")
	}
	if got := rr.Body.String(); got != "4375120948345356249" {
		t.Fatalf("body = %q, want challenge string", got)
	}
}

func TestMessagePushHandlerRejectsInvalidSignature(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/wechat/message-push?signature=invalid&timestamp=1714036504&nonce=1514711492&echostr=secret-challenge",
		nil,
	)
	rr := httptest.NewRecorder()

	MessagePushHandler("AAAAA").ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
	if got := rr.Body.String(); got == "secret-challenge" {
		t.Fatal("response must not echo challenge for an invalid signature")
	}
}

func TestMessagePushHandlerRejectsMissingVerificationParameter(t *testing.T) {
	for _, tc := range []struct {
		name string
		url  string
	}{
		{name: "signature", url: "/api/wechat/message-push?timestamp=1714036504&nonce=1514711492&echostr=secret-challenge"},
		{name: "timestamp", url: "/api/wechat/message-push?signature=invalid&nonce=1514711492&echostr=secret-challenge"},
		{name: "nonce", url: "/api/wechat/message-push?signature=invalid&timestamp=1714036504&echostr=secret-challenge"},
		{name: "echostr", url: "/api/wechat/message-push?signature=f464b24fc39322e44b38aa78f5edd27bd1441696&timestamp=1714036504&nonce=1514711492"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rr := httptest.NewRecorder()

			MessagePushHandler("AAAAA").ServeHTTP(rr, req)

			if rr.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
			}
			if got := rr.Body.String(); got == "secret-challenge" {
				t.Fatal("response must not echo challenge when a verification parameter is missing")
			}
		})
	}
}

func TestMessagePushHandlerRejectsMissingToken(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/wechat/message-push?signature=f464b24fc39322e44b38aa78f5edd27bd1441696&timestamp=1714036504&nonce=1514711492&echostr=secret-challenge",
		nil,
	)
	rr := httptest.NewRecorder()

	MessagePushHandler("").ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
	if got := rr.Body.String(); got == "secret-challenge" {
		t.Fatal("response must not echo challenge when Token is missing")
	}
}

func TestMessagePushHandlerReturnsChallengeWithTokenOnly(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/wechat/message-push?signature=f464b24fc39322e44b38aa78f5edd27bd1441696&timestamp=1714036504&nonce=1514711492&echostr=secret-challenge",
		nil,
	)
	rr := httptest.NewRecorder()

	MessagePushHandler("AAAAA").ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Body.String(); got != "secret-challenge" {
		t.Fatalf("body = %q, want challenge string", got)
	}
}
