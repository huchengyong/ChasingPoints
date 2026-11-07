package public

import (
	"net/http"

	"chasing_points/internal/logic/public"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

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

		l := public.NewGetLeaderboardLogic(r.Context(), svcCtx)
		resp, err := l.GetLeaderboard(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
