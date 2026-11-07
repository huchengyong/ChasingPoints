package ws

import (
	"net/http/httptest"
	"testing"

	"chasing_points/internal/config"
)

func TestWebSocketOriginAllowsConfiguredOrigin(t *testing.T) {
	cfg := config.Config{}
	cfg.Security.WebSocketAllowedOrigins = "https://app.example.com,https://mp.example.com"

	req := httptest.NewRequest("GET", "http://api.example.com/api/user/ws", nil)
	req.Header.Set("Origin", "https://mp.example.com")

	if !isWebSocketOriginAllowed(cfg, req) {
		t.Fatal("expected configured origin to be allowed")
	}
}

func TestWebSocketOriginRejectsUnconfiguredOrigin(t *testing.T) {
	cfg := config.Config{}
	cfg.Security.WebSocketAllowedOrigins = "https://app.example.com"

	req := httptest.NewRequest("GET", "http://api.example.com/api/user/ws", nil)
	req.Header.Set("Origin", "https://evil.example.com")

	if isWebSocketOriginAllowed(cfg, req) {
		t.Fatal("expected unconfigured origin to be rejected")
	}
}

func TestWebSocketOriginAllowsNativeMissingOriginOnlyWhenSecureAndEnabled(t *testing.T) {
	cfg := config.Config{}
	req := httptest.NewRequest("GET", "http://api.example.com/api/user/ws", nil)

	if isWebSocketOriginAllowed(cfg, req) {
		t.Fatal("expected missing origin to be rejected by default")
	}

	cfg.Security.AllowNativeMissingOrigin = true
	if isWebSocketOriginAllowed(cfg, req) {
		t.Fatal("expected insecure missing origin to be rejected")
	}

	req.Header.Set("X-Forwarded-Proto", "https")
	if !isWebSocketOriginAllowed(cfg, req) {
		t.Fatal("expected secure native missing origin to be allowed when enabled")
	}
}
