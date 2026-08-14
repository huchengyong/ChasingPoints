package public

import (
	"net/http"

	"chasing_points/internal/logic/public"
	"chasing_points/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 获取静态段位配置
func GetRankConfigsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := public.NewGetRankConfigsLogic(r.Context(), svcCtx)
		resp, err := l.GetRankConfigs()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
