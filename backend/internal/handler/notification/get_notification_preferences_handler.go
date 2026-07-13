package notification

import (
	"net/http"

	"chasing_points/internal/logic/notification"
	"chasing_points/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 获取通知偏好
func GetNotificationPreferencesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := notification.NewGetNotificationPreferencesLogic(r.Context(), svcCtx)
		resp, err := l.GetNotificationPreferences()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
