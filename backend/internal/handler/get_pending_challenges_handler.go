package handler

import (
	"net/http"

	"billiard_master/internal/logic"
	"billiard_master/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 获取待处理挑战
func GetPendingChallengesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewGetPendingChallengesLogic(r.Context(), svcCtx)
		resp, err := l.GetPendingChallenges()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
