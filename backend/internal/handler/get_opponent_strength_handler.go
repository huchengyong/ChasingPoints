package handler

import (
	"net/http"

	"billiard_master/internal/logic"
	"billiard_master/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 强弱对手分析
func GetOpponentStrengthHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewGetOpponentStrengthLogic(r.Context(), svcCtx)
		resp, err := l.GetOpponentStrength()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
