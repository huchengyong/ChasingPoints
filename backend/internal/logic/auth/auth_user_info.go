package auth

import (
	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func buildAuthUserInfo(user *model.User) *types.UserInfo {
	if user == nil {
		return nil
	}

	maskedPhone := ""
	if user.Phone != nil && len(*user.Phone) == 11 {
		maskedPhone = (*user.Phone)[:3] + "****" + (*user.Phone)[7:]
	}

	return &types.UserInfo{
		Id:        user.Id,
		Phone:     maskedPhone,
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}
