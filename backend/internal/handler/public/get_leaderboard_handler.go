package public

import (
	"context"
	"net/http"
	"strings"

	"chasing_points/internal/logic/public"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/golang-jwt/jwt/v4"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 获取段位排行榜
func GetLeaderboardHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetLeaderboardReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 尝试从 Authorization header 解析用户ID（公开接口可选登录）
		ctx := r.Context()
		if authHeader := r.Header.Get("Authorization"); authHeader != "" {
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenString := strings.TrimPrefix(authHeader, "Bearer ")
				if userId := parseUserIdFromToken(tokenString, svcCtx.Config.Auth.AccessSecret); userId > 0 {
					ctx = context.WithValue(ctx, "user_id", userId)
				}
			}
		}

		l := public.NewGetLeaderboardLogic(ctx, svcCtx)
		resp, err := l.GetLeaderboard(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

// parseUserIdFromToken 从JWT token中解析用户ID
func parseUserIdFromToken(tokenString, secret string) int64 {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return 0
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if userIdFloat, ok := claims["user_id"].(float64); ok {
			return int64(userIdFloat)
		}
	}

	return 0
}
