package feedback

import (
	"net/http"

	"chasing_points/internal/logic/feedback"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 公开提交投诉举报与意见反馈
func CreateFeedbackTicketHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CreateFeedbackTicketReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := feedback.NewCreateFeedbackTicketLogic(r.Context(), svcCtx)
		resp, err := l.CreateFeedbackTicket(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
