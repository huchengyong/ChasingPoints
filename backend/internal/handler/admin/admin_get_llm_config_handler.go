package admin

import (
	"net/http"

	"chasing_points/internal/logic/admin"
	"chasing_points/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 获取大模型调用配置
func AdminGetLlmConfigHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := admin.NewAdminGetLlmConfigLogic(r.Context(), svcCtx)
		resp, err := l.AdminGetLlmConfig()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
