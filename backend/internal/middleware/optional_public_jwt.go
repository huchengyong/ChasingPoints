package middleware

import (
	"context"
	"net/http"
	"strings"

	"chasing_points/internal/pkg"

	jwt "github.com/golang-jwt/jwt/v5"
)

// OptionalPublicJWTMiddleware accepts anonymous public reads while preserving a
// valid signed access identity for personalized public responses.
type OptionalPublicJWTMiddleware struct {
	accessSecret string
}

func NewOptionalPublicJWTMiddleware(accessSecret string) *OptionalPublicJWTMiddleware {
	return &OptionalPublicJWTMiddleware{accessSecret: accessSecret}
}

func (m *OptionalPublicJWTMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/public/") || r.Context().Value("user_id") != nil {
			next(w, r)
			return
		}
		authorization := strings.TrimSpace(r.Header.Get("Authorization"))
		if authorization == "" {
			next(w, r)
			return
		}
		if m == nil || strings.TrimSpace(m.accessSecret) == "" {
			http.Error(w, "invalid access token", http.StatusUnauthorized)
			return
		}
		tokenText, ok := strings.CutPrefix(authorization, "Bearer ")
		if !ok || strings.TrimSpace(tokenText) == "" {
			http.Error(w, "invalid access token", http.StatusUnauthorized)
			return
		}
		claims := &pkg.JwtClaims{}
		token, err := jwt.ParseWithClaims(tokenText, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(m.accessSecret), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		if err != nil || token == nil || !token.Valid || claims.UserId <= 0 || (claims.TokenType != "" && claims.TokenType != pkg.AccessTokenType) {
			http.Error(w, "invalid access token", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), "user_id", claims.UserId)
		if claims.TokenType != "" {
			ctx = context.WithValue(ctx, "token_type", claims.TokenType)
		}
		next(w, r.WithContext(ctx))
	}
}
