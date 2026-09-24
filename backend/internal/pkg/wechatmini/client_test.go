package wechatmini

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPClientExchangeLoginCodeReturnsOnlyIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sns/jscode2session" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("appid"); got != "mini-app-id" {
			t.Fatalf("unexpected appid: %s", got)
		}
		if got := r.URL.Query().Get("js_code"); got != "login-code" {
			t.Fatalf("unexpected login code: %s", got)
		}
		_, _ = io.WriteString(w, `{"openid":"openid-1","unionid":"unionid-1","session_key":"must-not-leak"}`)
	}))
	defer server.Close()

	client := NewHTTPClient("mini-app-id", "mini-app-secret", time.Second)
	client.loginEndpoint = server.URL + "/sns/jscode2session"

	identity, err := client.ExchangeLoginCode(context.Background(), "login-code")
	if err != nil {
		t.Fatalf("exchange login code: %v", err)
	}
	if identity.OpenID != "openid-1" || identity.UnionID != "unionid-1" {
		t.Fatalf("unexpected identity: %#v", identity)
	}
}

func TestHTTPClientGetPhoneNumberCachesAccessToken(t *testing.T) {
	accessTokenCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cgi-bin/token":
			accessTokenCalls++
			_, _ = io.WriteString(w, `{"access_token":"server-access-token","expires_in":7200}`)
		case "/wxa/business/getuserphonenumber":
			if got := r.URL.Query().Get("access_token"); got != "server-access-token" {
				t.Fatalf("unexpected access token: %s", got)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			if !strings.Contains(string(body), `"code":"phone-code"`) {
				t.Fatalf("unexpected request body: %s", body)
			}
			_, _ = io.WriteString(w, `{"errcode":0,"phone_info":{"phoneNumber":"13800138000"}}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewHTTPClient("mini-app-id", "mini-app-secret", time.Second)
	client.accessTokenEndpoint = server.URL + "/cgi-bin/token"
	client.phoneEndpoint = server.URL + "/wxa/business/getuserphonenumber"

	for i := 0; i < 2; i++ {
		phone, err := client.GetPhoneNumber(context.Background(), "phone-code")
		if err != nil {
			t.Fatalf("get phone number: %v", err)
		}
		if phone != "13800138000" {
			t.Fatalf("unexpected phone: %s", phone)
		}
	}

	if accessTokenCalls != 1 {
		t.Fatalf("expected one access token call, got %d", accessTokenCalls)
	}
}

func TestHTTPClientRejectsMissingCredentials(t *testing.T) {
	client := NewHTTPClient("", "", time.Second)

	_, err := client.ExchangeLoginCode(context.Background(), "login-code")
	if !IsErrorKind(err, ErrorKindUnavailable) {
		t.Fatalf("expected unavailable error, got %v", err)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatalf("credential error leaked secret detail: %v", err)
	}
}

func TestHTTPClientMapsWechatErrorsWithoutLeakingRawMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"errcode":40029,"errmsg":"jscode is invalid and secret detail"}`)
	}))
	defer server.Close()

	client := NewHTTPClient("mini-app-id", "mini-app-secret", time.Second)
	client.loginEndpoint = server.URL

	_, err := client.ExchangeLoginCode(context.Background(), "login-code")
	if !IsErrorKind(err, ErrorKindRejected) {
		t.Fatalf("expected rejected error, got %v", err)
	}
	if strings.Contains(err.Error(), "secret detail") || strings.Contains(err.Error(), "jscode") {
		t.Fatalf("wechat raw error leaked to caller: %v", err)
	}
}

func TestHTTPClientMapsNetworkFailuresToTemporaryError(t *testing.T) {
	client := NewHTTPClient("mini-app-id", "mini-app-secret", time.Second)
	client.loginEndpoint = "http://127.0.0.1:1/unavailable"

	_, err := client.ExchangeLoginCode(context.Background(), "login-code")
	if !IsErrorKind(err, ErrorKindTemporary) {
		t.Fatalf("expected temporary error, got %v", err)
	}
}

func TestHTTPClientMapsTimeoutToTemporaryError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		time.Sleep(20 * time.Millisecond)
	}))
	defer server.Close()

	client := NewHTTPClient("mini-app-id", "mini-app-secret", time.Millisecond)
	client.loginEndpoint = server.URL

	_, err := client.ExchangeLoginCode(context.Background(), "login-code")
	if !IsErrorKind(err, ErrorKindTemporary) {
		t.Fatalf("expected timeout to become a temporary error, got %v", err)
	}
}
