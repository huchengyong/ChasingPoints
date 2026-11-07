package challenge

import (
	"net/http"

	"chasing_points/internal/logic/challenge"
	"chasing_points/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 当前约球摘要
func GetChallengeSummaryHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := challenge.NewGetChallengeSummaryLogic(r.Context(), svcCtx)
		resp, err := l.GetChallengeSummary()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
