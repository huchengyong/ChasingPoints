package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminUpdateUserStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新用户状态
func NewAdminUpdateUserStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminUpdateUserStatusLogic {
	return &AdminUpdateUserStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminUpdateUserStatusLogic) AdminUpdateUserStatus(req *types.AdminUserUpdateStatusReq) (resp *types.AdminWriteResp, err error) {
	// 验证参数
	if req.UserId <= 0 {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "用户ID无效",
		}, nil
	}
	if req.Status != 0 && req.Status != 1 {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "状态值无效",
		}, nil
	}

	// 查询用户是否存在
	user, err := l.svcCtx.UserModel.FindById(req.UserId)
	if err != nil {
		l.Logger.Errorf("查询用户失败: userId=%d err=%v", req.UserId, err)
		return &types.AdminWriteResp{
			Code:    500,
			Success: false,
			Message: "系统错误",
		}, nil
	}
	if user == nil {
		return &types.AdminWriteResp{
			Code:    404,
			Success: false,
			Message: "用户不存在",
		}, nil
	}

	// 更新状态
	if err := l.svcCtx.UserModel.UpdateStatus(req.UserId, req.Status); err != nil {
		l.Logger.Errorf("更新用户状态失败: userId=%d status=%d err=%v", req.UserId, req.Status, err)
		return &types.AdminWriteResp{
			Code:    500,
			Success: false,
			Message: "更新失败",
		}, nil
	}

	var msg string
	if req.Status == 1 {
		msg = "用户已启用"
	} else {
		msg = "用户已禁用"
	}

	return &types.AdminWriteResp{
		Code:    0,
		Success: true,
		Message: msg,
	}, nil
}
