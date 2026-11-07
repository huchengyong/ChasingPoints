package match

import (
	"net/http"

	"chasing_points/internal/logic/match"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 更新未记分斯诺克对局赛制
func UpdateSnookerFormatHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdateSnookerFormatReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := match.NewUpdateSnookerFormatLogic(r.Context(), svcCtx)
		resp, err := l.UpdateSnookerFormat(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
