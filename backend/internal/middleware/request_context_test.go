package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestContextMiddlewareRedactsCredentialsAfterRequest(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer secret")
	request.Header.Set("Cookie", "session=secret")

	RequestContextMiddleware(func(_ http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer secret" || request.Header.Get("Cookie") != "session=secret" {
			t.Fatal("credentials must remain available during request handling")
		}
	})(httptest.NewRecorder(), request)

	if request.Header.Get("Authorization") != "" || request.Header.Get("Cookie") != "" {
		t.Fatal("credentials must be removed before outer request logging")
	}
}
