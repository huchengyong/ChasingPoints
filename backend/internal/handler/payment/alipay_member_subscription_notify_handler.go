package payment

import (
	"net/http"

	"chasing_points/internal/logic/payment"
	"chasing_points/internal/svc"
)

// 支付宝会员支付异步回调
func AlipayMemberSubscriptionNotifyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := payment.NewAlipayMemberSubscriptionNotifyLogic(r.Context(), svcCtx)
		resp, _ := l.AlipayMemberSubscriptionNotify()

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if resp != nil && resp.Success {
			_, _ = w.Write([]byte("success"))
			return
		}
		_, _ = w.Write([]byte("failure"))
	}
}
