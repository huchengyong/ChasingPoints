package notification

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

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
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetNotificationListLogic) GetNotificationList(req *types.GetNotificationListReq) (resp *types.GetNotificationListResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetNotificationListResp{Success: false}, nil
	}

	if l.svcCtx == nil || l.svcCtx.NotificationModel == nil {
		return &types.GetNotificationListResp{Success: false}, nil
	}
	if req == nil {
		req = &types.GetNotificationListReq{}
	}
	list, total, unreadCount, err := l.svcCtx.NotificationModel.FindPageWithUnreadCount(userIdInt, req.Page, req.PageSize, req.Type)
	if err != nil {
		l.Logger.Errorf("查询通知列表失败: %v", err)
		return &types.GetNotificationListResp{Success: false}, nil
	}

	items := make([]types.NotificationInfo, 0, len(list))
	for _, item := range list {
		data := ""
		if item.Data != nil {
			data = *item.Data
		}

		items = append(items, types.NotificationInfo{
			Id:        item.Id,
			Type:      item.Type,
			Title:     item.Title,
			Content:   item.Content,
			Data:      data,
			IsRead:    item.IsRead == 1,
			CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &types.GetNotificationListResp{
		Success:     true,
		Total:       total,
		UnreadCount: int(unreadCount),
		List:        items,
	}, nil
}
