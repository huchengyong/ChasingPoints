package notification

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteNotificationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 删除通知
func NewDeleteNotificationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteNotificationLogic {
	return &DeleteNotificationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteNotificationLogic) DeleteNotification(req *types.NotificationIdReq) (resp *types.CommonResp, err error) {
	// todo: add your logic here and delete this line

	return
}
