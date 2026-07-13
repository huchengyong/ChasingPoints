package season

import (
	"net/http"

	"chasing_points/internal/logic/season"
	"chasing_points/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 获取当前赛季
func GetCurrentSeasonHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := season.NewGetCurrentSeasonLogic(r.Context(), svcCtx)
		resp, err := l.GetCurrentSeason()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
