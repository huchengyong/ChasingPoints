package notification

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetNotificationListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取通知列表
func NewGetNotificationListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetNotificationListLogic {
	return &GetNotificationListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetNotificationListLogic) GetNotificationList(req *types.GetNotificationListReq) (resp *types.GetNotificationListResp, err error) {
	// todo: add your logic here and delete this line

	return
}
