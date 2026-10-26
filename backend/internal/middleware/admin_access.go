package middleware

import (
	"net/http"
	"path"
	"strings"

	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

const adminAPIPrefix = "/api/admin"

// adminPublicPaths are the only /api/admin entries that may be called without
// an administrator identity. Anything new under /api/admin is protected until
// it is added here on purpose.
var adminPublicPaths = map[string]bool{
	"/api/admin/login":  true,
	"/api/admin/init":   true,
	"/api/admin/exists": true,
}

type adminAccessPayload struct {
	Code    int    `json:"code"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// AdminAccessMiddleware keeps every /api/admin route behind an administrator
// identity. Administrator and user tokens share one signing secret and differ
// only by the sign of the user id (admins are negative), so go-zero's JWT check
// alone lets any logged-in user reach an admin handler that forgets to check.
// It runs after the JWT handler has put user_id into the context.
type AdminAccessMiddleware struct{}

func NewAdminAccessMiddleware() *AdminAccessMiddleware {
	return &AdminAccessMiddleware{}
}

func (m *AdminAccessMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// go-zero's router cleans the path before matching, so `/api//admin/x`
		// and `/api/admin/./x` reach admin handlers; judge the same cleaned path.
		cleaned := path.Clean(r.URL.Path)
		if !isAdminAPIPath(cleaned) || adminPublicPaths[cleaned] {
			next(w, r)
			return
		}

		userID, err := utils.GetOptionalSignedUserIDFromCtx(r.Context())
		if err != nil {
			logx.WithContext(r.Context()).Errorf("解析后台请求身份失败: %v", err)
			userID = 0
		}
		switch {
		case userID < 0:
			next(w, r)
		case userID == 0:
			httpx.WriteJsonCtx(r.Context(), w, http.StatusUnauthorized, adminAccessPayload{Code: http.StatusUnauthorized, Message: "未登录或登录已过期"})
		default:
			logx.WithContext(r.Context()).Infof("普通用户令牌访问后台接口被拒绝: user_id=%d path=%s", userID, cleaned)
			httpx.WriteJsonCtx(r.Context(), w, http.StatusForbidden, adminAccessPayload{Code: http.StatusForbidden, Message: "无管理员权限"})
		}
	}
}

func isAdminAPIPath(cleaned string) bool {
	return cleaned == adminAPIPrefix || strings.HasPrefix(cleaned, adminAPIPrefix+"/")
}
