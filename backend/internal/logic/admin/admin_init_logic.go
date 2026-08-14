package admin

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminInitLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 初始化管理员账号
func NewAdminInitLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminInitLogic {
	return &AdminInitLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *AdminInitLogic) AdminInit(req *types.AdminInitReq) (resp *types.AdminInitResp, err error) {
	if l.svcCtx.Config.Admin.SetupToken == "" {
		return &types.AdminInitResp{
			Code:    503,
			Success: false,
			Message: "管理员初始化未启用，请联系部署人员配置",
		}, nil
	}

	if req.SetupToken != l.svcCtx.Config.Admin.SetupToken {
		return &types.AdminInitResp{
			Code:    403,
			Success: false,
			Message: "初始化凭证无效",
		}, nil
	}

	normalizedEmail := normalizeAdminEmail(req.Email)
	if !validateAdminEmail(normalizedEmail) {
		return &types.AdminInitResp{
			Code:    400,
			Success: false,
			Message: "请输入有效的管理员邮箱",
		}, nil
	}

	if !validateAdminPassword(req.Password) {
		return &types.AdminInitResp{
			Code:    400,
			Success: false,
			Message: "管理员密码长度需为8-72位",
		}, nil
	}

	// 检查是否已有管理员
	count, err := l.svcCtx.AdminModel.Count(l.ctx)
	if err != nil {
		l.Logger.Errorf("检查管理员数量失败: %v", err)
		return &types.AdminInitResp{
			Code:    500,
			Success: false,
			Message: "系统错误",
		}, nil
	}

	if count > 0 {
		return &types.AdminInitResp{
			Code:    400,
			Success: false,
			Message: "管理员账号已存在，请勿重复初始化",
		}, nil
	}

	// 密码加密
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		l.Logger.Errorf("密码加密失败: %v", err)
		return &types.AdminInitResp{
			Code:    500,
			Success: false,
			Message: "系统错误",
		}, nil
	}

	// 创建管理员
	admin := &model.Admin{
		Email:    normalizedEmail,
		Password: hashedPassword,
		Nickname: "超级管理员",
		Role:     "super_admin",
		Status:   1,
	}

	err = l.svcCtx.AdminModel.Create(l.ctx, admin)
	if err != nil {
		l.Logger.Errorf("创建管理员失败: %v", err)
		return &types.AdminInitResp{
			Code:    500,
			Success: false,
			Message: "系统错误",
		}, nil
	}

	return &types.AdminInitResp{
		Code:    0,
		Success: true,
		Message: "管理员账号初始化成功",
	}, nil
}
