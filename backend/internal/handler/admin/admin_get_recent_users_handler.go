package admin

import (
	"net/http"

	"chasing_points/internal/logic"
	"chasing_points/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 获取最近注册用户
func AdminGetRecentUsersHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewAdminGetRecentUsersLogic(r.Context(), svcCtx)
		resp, err := l.AdminGetRecentUsers()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
