package admin

import (
	"net/http"

	"chasing_points/internal/logic/admin"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 创建赛事阶段
func AdminCreateEventNewsStageHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AdminEventNewsStageCreateReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := admin.NewAdminCreateEventNewsStageLogic(r.Context(), svcCtx)
		resp, err := l.AdminCreateEventNewsStage(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
