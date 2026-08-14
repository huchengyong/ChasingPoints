package user

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateUserPrivacyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新用户隐私设置
func NewUpdateUserPrivacyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserPrivacyLogic {
	return &UpdateUserPrivacyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *UpdateUserPrivacyLogic) UpdateUserPrivacy(req *types.UpdateUserPrivacyReq) (resp *types.UpdateUserPrivacyResp, err error) {
	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.UpdateUserPrivacyResp{
			Success: false,
			Message: "更新隐私设置失败",
		}, nil
	}

	user, err := l.svcCtx.UserModel.FindById(userID)
	if err != nil {
		l.Logger.Errorf("查询用户失败: userId=%d err=%v", userID, err)
		return &types.UpdateUserPrivacyResp{
			Success: false,
			Message: "更新隐私设置失败",
		}, nil
	}
	if user == nil {
		return &types.UpdateUserPrivacyResp{
			Success: false,
			Message: "用户不存在",
		}, nil
	}

	if err := l.svcCtx.UserModel.UpdateHideMatchRecord(userID, req.HideMatchRecord); err != nil {
		l.Logger.Errorf("更新用户隐私设置失败: userId=%d err=%v", userID, err)
		return &types.UpdateUserPrivacyResp{
			Success: false,
			Message: "更新隐私设置失败",
		}, nil
	}

	return &types.UpdateUserPrivacyResp{
		Success:         true,
		Message:         "更新成功",
		HideMatchRecord: req.HideMatchRecord,
	}, nil
}
