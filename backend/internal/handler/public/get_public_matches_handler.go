package public

import (
	"context"
	"net/http"
	"strings"

	"chasing_points/internal/logic/public"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 获取公开观赛对局列表
func GetPublicMatchesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.PublicMatchListReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		ctx := r.Context()
		if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if userId := parseUserIdFromToken(tokenString, svcCtx.Config.Auth.AccessSecret); userId > 0 {
				ctx = context.WithValue(ctx, "user_id", userId)
			}
		}

		l := public.NewGetPublicMatchesLogic(ctx, svcCtx)
		resp, err := l.GetPublicMatches(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
