package gear

import (
	"net/http"

	"chasing_points/internal/logic/gear"
	"chasing_points/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 获取我的装备与主力设置
func GetGearMineHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := gear.NewGetGearMineLogic(r.Context(), svcCtx)
		resp, err := l.GetGearMine()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
