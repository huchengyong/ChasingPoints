package public

import (
	"net/http"

	"chasing_points/internal/logic/public"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 未登录预览匹配邀请（凭短码）
func PreviewPublicMatchInviteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.PublicMatchInvitePreviewReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := public.NewPreviewPublicMatchInviteLogic(r.Context(), svcCtx)
		resp, err := l.PreviewPublicMatchInvite(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
