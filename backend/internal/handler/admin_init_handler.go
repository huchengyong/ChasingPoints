package handler

import (
	"net/http"

	"billiard_master/internal/logic"
	"billiard_master/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 初始化管理员账号
func AdminInitHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := parseAdminInitRequest(r)
		if err != nil {
			writeAdminParseError(r, w, err)
			return
		}

		l := logic.NewAdminInitLogic(r.Context(), svcCtx)
		resp, err := l.AdminInit(req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
