package match

import (
	"net/http"

	"chasing_points/internal/logic/match"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 裁判码预览
func RefereePreviewHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RefereePreviewReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		l := match.NewRefereePreviewLogic(r.Context(), svcCtx)
		resp, err := l.RefereePreview(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
