package auth

import "chasing_points/internal/types"

func buildBindPhoneSuccessResp() *types.BindPhoneResp {
	return &types.BindPhoneResp{
		Success:       true,
		Message:       "绑定成功",
		MergedAccount: false,
		NeedBindPhone: false,
	}
}
