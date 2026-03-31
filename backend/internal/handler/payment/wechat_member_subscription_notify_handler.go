package payment

import (
	"encoding/json"
	"net/http"

	"chasing_points/internal/logic/payment"
	"chasing_points/internal/svc"
)

// 微信会员支付异步回调
func WechatMemberSubscriptionNotifyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := payment.NewWechatMemberSubscriptionNotifyLogic(r.Context(), svcCtx)
		resp, _ := l.WechatMemberSubscriptionNotify()

		result := map[string]string{
			"code":    "FAIL",
			"message": "失败",
		}
		if resp != nil && resp.Success {
			result["code"] = "SUCCESS"
			result["message"] = "成功"
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(result)
	}
}
