package middleware

import (
	"net/http"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

const (
	// SessionInvalidReason marks an access token whose subject no longer maps
	// to an enabled user, as opposed to an ordinary expired JWT.
	SessionInvalidReason     = "SESSION_INVALID"
	InvalidAccessTokenReason = "INVALID_ACCESS_TOKEN"

	sessionInvalidMessage     = "登录状态已失效，请重新登录"
	invalidAccessTokenMessage = "无效的访问令牌"
)

type sessionInvalidPayload struct {
	Success bool   `json:"success"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// ActiveUserSessionMiddleware rejects REST requests whose JWT identifies a
// regular user that no longer exists or is disabled. It runs after go-zero's
// JWT middleware has populated user_id in the context, so it only reads the
// signed identity value: anonymous requests and negative admin identities
// keep their existing behavior.
type ActiveUserSessionMiddleware struct {
	userModel *model.UserModel
}

func NewActiveUserSessionMiddleware(userModel *model.UserModel) *ActiveUserSessionMiddleware {
	return &ActiveUserSessionMiddleware{userModel: userModel}
}

func (m *ActiveUserSessionMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := utils.GetOptionalSignedUserIDFromCtx(r.Context())
		if err != nil {
			logx.WithContext(r.Context()).Errorf("解析用户身份失败: %v", err)
			http.Error(w, "内部服务错误", http.StatusInternalServerError)
			return
		}
		// Anonymous requests and admin identities keep their existing behavior.
		if userID <= 0 {
			next(w, r)
			return
		}

		tokenType, err := utils.GetOptionalTokenTypeFromCtx(r.Context())
		if err != nil {
			logx.WithContext(r.Context()).Errorf("解析令牌类型失败: %v", err)
			http.Error(w, "内部服务错误", http.StatusInternalServerError)
			return
		}
		// Legacy access tokens have no token_type; explicitly typed non-access
		// tokens must never reach business handlers.
		if tokenType != "" && tokenType != pkg.AccessTokenType {
			httpx.WriteJsonCtx(r.Context(), w, http.StatusUnauthorized, sessionInvalidPayload{
				Success: false,
				Reason:  InvalidAccessTokenReason,
				Message: invalidAccessTokenMessage,
			})
			return
		}

		user, err := m.userModel.FindById(userID)
		if err != nil {
			logx.WithContext(r.Context()).Errorf("校验用户 %d 失败: %v", userID, err)
			http.Error(w, "内部服务错误", http.StatusInternalServerError)
			return
		}
		if user == nil || user.Status != 1 {
			httpx.WriteJsonCtx(r.Context(), w, http.StatusUnauthorized, sessionInvalidPayload{
				Success: false,
				Reason:  SessionInvalidReason,
				Message: sessionInvalidMessage,
			})
			return
		}

		next(w, r)
	}
}
