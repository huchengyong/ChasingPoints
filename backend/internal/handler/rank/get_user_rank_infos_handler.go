package rank

import (
	"net/http"

	"chasing_points/internal/logic/rank"
	"chasing_points/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 获取用户全部球种段位信息
func GetUserRankInfosHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := rank.NewGetUserRankInfosLogic(r.Context(), svcCtx)
		resp, err := l.GetUserRankInfos()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
