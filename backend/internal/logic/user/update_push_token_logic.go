package user

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdatePushTokenLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新推送令牌
func NewUpdatePushTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdatePushTokenLogic {
	return &UpdatePushTokenLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *UpdatePushTokenLogic) UpdatePushToken(req *types.UpdatePushTokenReq) (resp *types.CommonResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CommonResp{Success: false, Message: "获取用户信息失败"}, nil
	}

	if req.PushClientId == "" {
		return &types.CommonResp{Success: false, Message: "推送令牌不能为空"}, nil
	}

	if err := l.svcCtx.UserModel.UpdatePushToken(userIdInt, req.PushClientId); err != nil {
		l.Logger.Errorf("更新推送令牌失败: userId=%d err=%v", userIdInt, err)
		return &types.CommonResp{Success: false, Message: "更新失败"}, nil
	}

	return &types.CommonResp{Success: true, Message: "更新成功"}, nil
}
