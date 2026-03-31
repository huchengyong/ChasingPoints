package member

import (
	"net/http"

	"chasing_points/internal/logic/member"
	"chasing_points/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 获取会员套餐列表
func GetMemberPlansHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := member.NewGetMemberPlansLogic(r.Context(), svcCtx)
		resp, err := l.GetMemberPlans()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
