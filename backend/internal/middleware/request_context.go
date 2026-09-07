package middleware

import (
	"chasing_points/internal/utils"
	"net/http"
)

// RequestContextMiddleware 将 request 信息添加到 context
func RequestContextMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			r.Header.Del("Authorization")
			r.Header.Del("Cookie")
		}()
		ctx := utils.WithRequest(r.Context(), r)
		next(w, r.WithContext(ctx))
	}
}
