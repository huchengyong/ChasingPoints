package logic

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminUserInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取当前管理员信息
func NewAdminUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminUserInfoLogic {
	return &AdminUserInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminUserInfoLogic) AdminUserInfo() (resp *types.AdminUserInfoResp, err error) {
	// 获取当前管理员ID
	adminId, err := utils.GetAdminIDFromCtx(l.ctx)
	if err != nil {
		return &types.AdminUserInfoResp{
			Code:    401,
			Success: false,
			Message: "未登录或登录已过期",
		}, nil
	}

	// 查询管理员信息
	admin, err := l.svcCtx.AdminModel.FindById(l.ctx, adminId)
	if err != nil {
		return &types.AdminUserInfoResp{
			Code:    500,
			Success: false,
			Message: "查询管理员信息失败",
		}, nil
	}

	return &types.AdminUserInfoResp{
		Code:    0,
		Success: true,
		Message: "success",
		UserInfo: &types.AdminInfo{
			Id:       int64(admin.Id),
			Email:    admin.Email,
			Nickname: admin.Nickname,
			Avatar:   admin.Avatar,
			Role:     admin.Role,
		},
	}, nil
}
