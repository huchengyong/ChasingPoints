package stats

import (
	"net/http"

	"chasing_points/internal/logic"
	"chasing_points/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 分球种统计
func GetStatsByGameTypeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewGetStatsByGameTypeLogic(r.Context(), svcCtx)
		resp, err := l.GetStatsByGameType()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
