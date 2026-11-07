package user

import (
	"net/http"

	"chasing_points/internal/logic/user"
	"chasing_points/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 签发用户级 WebSocket 连接 ticket
func IssueUserWSTicketHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := user.NewIssueUserWSTicketLogic(r.Context(), svcCtx)
		resp, err := l.IssueUserWSTicket()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
