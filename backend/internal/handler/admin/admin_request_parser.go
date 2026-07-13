package admin

import (
	"encoding/json"
	"fmt"
	"net/http"

	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

type adminInitRequestPayload struct {
	Email           string `json:"email"`
	Password        string `json:"password"`
	SetupToken      string `json:"setup_token"`
	SetupTokenCamel string `json:"setupToken"`
}

type adminChangePasswordPayload struct {
	OldPassword      string `json:"old_password"`
	OldPasswordCamel string `json:"oldPassword"`
	NewPassword      string `json:"new_password"`
	NewPasswordCamel string `json:"newPassword"`
}

func parseAdminInitRequest(r *http.Request) (*types.AdminInitReq, error) {
	var payload adminInitRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("请求体不是合法JSON: %w", err)
	}

	setupToken := payload.SetupToken
	if setupToken == "" {
		setupToken = payload.SetupTokenCamel
	}

	return &types.AdminInitReq{
		Email:      payload.Email,
		Password:   payload.Password,
		SetupToken: setupToken,
	}, nil
}

func parseAdminChangePasswordRequest(r *http.Request) (*types.AdminChangePasswordReq, error) {
	var payload adminChangePasswordPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("请求体不是合法JSON: %w", err)
	}

	oldPassword := payload.OldPassword
	if oldPassword == "" {
		oldPassword = payload.OldPasswordCamel
	}

	newPassword := payload.NewPassword
	if newPassword == "" {
		newPassword = payload.NewPasswordCamel
	}

	return &types.AdminChangePasswordReq{
		OldPassword: oldPassword,
		NewPassword: newPassword,
	}, nil
}

func writeAdminParseError(r *http.Request, w http.ResponseWriter, err error) {
	httpx.WriteJsonCtx(r.Context(), w, http.StatusBadRequest, map[string]any{
		"code":    400,
		"success": false,
		"message": err.Error(),
	})
}
