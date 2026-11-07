package admin

import (
	"net/http"

	"chasing_points/internal/logic/admin"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 保存装备品牌（管理员）
func AdminSaveGearBrandHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AdminGearBrandSaveReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := admin.NewAdminSaveGearBrandLogic(r.Context(), svcCtx)
		resp, err := l.AdminSaveGearBrand(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
