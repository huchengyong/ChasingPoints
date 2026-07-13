package user

import (
	"net/http"

	"chasing_points/internal/logic/user"
	"chasing_points/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 获取当前用户信誉状态
func GetUserReputationHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := user.NewGetUserReputationLogic(r.Context(), svcCtx)
		resp, err := l.GetUserReputation()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
