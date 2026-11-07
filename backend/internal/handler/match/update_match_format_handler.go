package match

import (
	"net/http"

	"chasing_points/internal/logic/match"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 更新未记分中八/美九对局赛制
func UpdateMatchFormatHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdateMatchFormatReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := match.NewUpdateMatchFormatLogic(r.Context(), svcCtx)
		resp, err := l.UpdateMatchFormat(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
