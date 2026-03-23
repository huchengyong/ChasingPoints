package handler

import (
	"net/http"

	"billiard_master/internal/logic"
	"billiard_master/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 获取已解锁成就
func GetUserAchievementsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewGetUserAchievementsLogic(r.Context(), svcCtx)
		resp, err := l.GetUserAchievements()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
