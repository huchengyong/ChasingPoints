package logic

import "billiard_master/internal/types"

func buildBindPhoneSuccessResp(mergedAccount bool) *types.BindPhoneResp {
	resp := &types.BindPhoneResp{
		Success:       true,
		MergedAccount: mergedAccount,
	}

	if mergedAccount {
		resp.Message = "账号已合并，请使用手机号登录"
		return resp
	}

	resp.Message = "绑定成功"
	return resp
}
