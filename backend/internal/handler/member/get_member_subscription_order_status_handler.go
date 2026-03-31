package member

import (
	"net/http"

	"chasing_points/internal/logic/member"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 查询会员订阅订单状态
func GetMemberSubscriptionOrderStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetMemberSubscriptionOrderStatusReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := member.NewGetMemberSubscriptionOrderStatusLogic(r.Context(), svcCtx)
		resp, err := l.GetMemberSubscriptionOrderStatus(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
