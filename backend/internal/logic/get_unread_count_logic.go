package logic

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUnreadCountLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取未读数量
func NewGetUnreadCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUnreadCountLogic {
	return &GetUnreadCountLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUnreadCountLogic) GetUnreadCount() (resp *types.GetUnreadCountResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetUnreadCountResp{Success: false}, nil
	}

	count, err := l.svcCtx.NotificationModel.GetUnreadCount(userIdInt)
	if err != nil {
		l.Logger.Errorf("查询未读通知数量失败: %v", err)
		return &types.GetUnreadCountResp{Success: false}, nil
	}

	return &types.GetUnreadCountResp{Success: true, Count: int(count)}, nil
}
