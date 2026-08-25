package ws

import (
	"net/http"
	"strings"

	"chasing_points/internal/config"

	"github.com/gorilla/websocket"
)

func newWebSocketUpgrader(c config.Config) *websocket.Upgrader {
	return &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return isWebSocketOriginAllowed(c, r)
		},
	}
}

func isWebSocketOriginAllowed(c config.Config, r *http.Request) bool {
	origin := strings.TrimRight(strings.TrimSpace(r.Header.Get("Origin")), "/")
	if origin == "" {
		return c.Security.AllowNativeMissingOrigin && requestUsesSecureTransport(r)
	}
	return config.OriginAllowed(origin, c.Security.WebSocketOriginList())
}

func requestUsesSecureTransport(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	if strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		return true
	}
	if strings.Contains(strings.ToLower(r.Header.Get("Forwarded")), "proto=https") {
		return true
	}
	return false
}
