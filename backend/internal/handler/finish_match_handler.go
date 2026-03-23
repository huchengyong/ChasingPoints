package handler

import (
	"net/http"

	"billiard_master/internal/logic"
	"billiard_master/internal/svc"
	"billiard_master/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 结束对局
func FinishMatchHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.FinishMatchReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewFinishMatchLogic(r.Context(), svcCtx)
		resp, err := l.FinishMatch(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
