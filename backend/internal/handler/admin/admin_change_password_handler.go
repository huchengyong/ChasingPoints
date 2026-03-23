package admin

import (
	"net/http"

	"chasing_points/internal/logic"
	"chasing_points/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 修改管理员密码
func AdminChangePasswordHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := parseAdminChangePasswordRequest(r)
		if err != nil {
			writeAdminParseError(r, w, err)
			return
		}

		l := logic.NewAdminChangePasswordLogic(r.Context(), svcCtx)
		resp, err := l.AdminChangePassword(req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
