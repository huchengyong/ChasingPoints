package utils

import (
	"context"
	"net/http"
	"strings"
)

// contextKey 用于在 context 中存储 request
type contextKey string

const RequestContextKey contextKey = "http_request"

// GetClientIP 从 context 中获取客户端 IP
func GetClientIP(ctx context.Context) string {
	req, ok := ctx.Value(RequestContextKey).(*http.Request)
	if !ok {
		return ""
	}

	// 优先从 X-Forwarded-For 获取
	xff := req.Header.Get("X-Forwarded-For")
	if xff != "" {
		// 取第一个 IP
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// 尝试 X-Real-Ip
	xri := req.Header.Get("X-Real-Ip")
	if xri != "" {
		return xri
	}

	// 从 RemoteAddr 获取
	remoteAddr := req.RemoteAddr
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		return remoteAddr[:idx]
	}
	return remoteAddr
}

// GetUserAgent 从 context 中获取 User-Agent
func GetUserAgent(ctx context.Context) string {
	req, ok := ctx.Value(RequestContextKey).(*http.Request)
	if !ok {
		return ""
	}
	return req.UserAgent()
}

// WithRequest 将 request 添加到 context
func WithRequest(ctx context.Context, req *http.Request) context.Context {
	return context.WithValue(ctx, RequestContextKey, req)
}
