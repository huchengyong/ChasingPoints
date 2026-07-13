package user

import (
	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func buildMaskedPhone(phone *string) string {
	if phone == nil {
		return ""
	}

	value := *phone
	if len(value) == 11 {
		return value[:3] + "****" + value[7:]
	}
	return value
}

func buildUserInfoPayload(user *model.User) *types.UserInfo {
	if user == nil {
		return nil
	}

	return &types.UserInfo{
		Id:        user.Id,
		Phone:     buildMaskedPhone(user.Phone),
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		Status:    user.Status,
		CreatedAt: logicx.FormatUTC8Time(user.CreatedAt),
	}
}
