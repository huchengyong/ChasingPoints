package eventnews

import (
	"net/http"

	"chasing_points/internal/logic/eventnews"
	"chasing_points/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 获取首页焦点赛事情报
func GetFeaturedEventNewsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := eventnews.NewGetFeaturedEventNewsLogic(r.Context(), svcCtx)
		resp, err := l.GetFeaturedEventNews()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
