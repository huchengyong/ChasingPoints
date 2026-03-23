package logic

import (
	"context"
	"strings"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminChangePasswordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 修改管理员密码
func NewAdminChangePasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminChangePasswordLogic {
	return &AdminChangePasswordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminChangePasswordLogic) AdminChangePassword(req *types.AdminChangePasswordReq) (resp *types.AdminWriteResp, err error) {
	adminId, err := utils.GetAdminIDFromCtx(l.ctx)
	if err != nil {
		return &types.AdminWriteResp{
			Code:    401,
			Success: false,
			Message: "未登录或登录已过期",
		}, nil
	}

	oldPassword := req.OldPassword
	newPassword := req.NewPassword
	if strings.TrimSpace(oldPassword) == "" {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "请输入当前密码",
		}, nil
	}
	if !validateAdminPassword(newPassword) {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "新密码长度需为8-72位",
		}, nil
	}
	if oldPassword == newPassword {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "新密码不能与当前密码相同",
		}, nil
	}

	admin, err := l.svcCtx.AdminModel.FindById(l.ctx, adminId)
	if err != nil {
		l.Logger.Errorf("查询管理员失败: %v", err)
		return &types.AdminWriteResp{
			Code:    500,
			Success: false,
			Message: "系统错误",
		}, nil
	}

	if !utils.CheckPasswordHash(oldPassword, admin.Password) {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "当前密码错误",
		}, nil
	}

	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		l.Logger.Errorf("加密新密码失败: %v", err)
		return &types.AdminWriteResp{
			Code:    500,
			Success: false,
			Message: "系统错误",
		}, nil
	}

	if err := l.svcCtx.AdminModel.UpdatePassword(l.ctx, adminId, hashedPassword); err != nil {
		l.Logger.Errorf("更新管理员密码失败: %v", err)
		return &types.AdminWriteResp{
			Code:    500,
			Success: false,
			Message: "系统错误",
		}, nil
	}

	return &types.AdminWriteResp{
		Code:    0,
		Success: true,
		Message: "密码修改成功",
	}, nil
}
