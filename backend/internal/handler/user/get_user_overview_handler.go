package user

import (
	"net/http"

	"chasing_points/internal/logic/user"
	"chasing_points/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 获取个人主页概览
func GetUserOverviewHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := user.NewGetUserOverviewLogic(r.Context(), svcCtx)
		resp, err := l.GetUserOverview()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
