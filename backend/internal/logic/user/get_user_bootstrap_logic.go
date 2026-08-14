package user

import (
	"context"

	matchlogic "chasing_points/internal/logic/match"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserBootstrapLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取应用前台恢复所需的用户活动快照
func NewGetUserBootstrapLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserBootstrapLogic {
	return &GetUserBootstrapLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetUserBootstrapLogic) GetUserBootstrap() (resp *types.GetUserBootstrapResp, err error) {
	result := &types.GetUserBootstrapResp{
		Availability:  map[string]bool{},
		PartialErrors: []types.ReadPartialError{},
	}
	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil || l.svcCtx == nil || l.svcCtx.UserModel == nil {
		return result, nil
	}
	user, err := currentUserFromRequest(l.ctx, l.svcCtx, userID)
	if err != nil || user == nil {
		return result, nil
	}
	result.Success = true
	result.UserInfo = buildUserInfoPayload(user)
	result.Availability["user"] = true

	if l.svcCtx.MatchModel == nil {
		appendBootstrapPartial(result, "current_match", "当前对局服务不可用")
	} else if current, currentErr := matchlogic.NewGetCurrentMatchLogic(l.ctx, l.svcCtx).GetCurrentMatch(); currentErr != nil || !current.Success {
		appendBootstrapPartial(result, "current_match", "当前对局读取失败")
	} else {
		result.CurrentMatch = current.Match
		result.Availability["current_match"] = true
	}

	if l.svcCtx.NotificationModel == nil {
		appendBootstrapPartial(result, "unread_count", "通知服务不可用")
		appendBootstrapPartial(result, "latest_season_rollover", "通知服务不可用")
	} else {
		if unreadCount, countErr := l.svcCtx.NotificationModel.GetUnreadCount(userID); countErr != nil {
			appendBootstrapPartial(result, "unread_count", "未读数读取失败")
		} else {
			result.UnreadCount = int(unreadCount)
			result.Availability["unread_count"] = true
		}
		if notification, notificationErr := l.svcCtx.NotificationModel.FindLatestUnreadByType(userID, "season_rollover"); notificationErr != nil {
			appendBootstrapPartial(result, "latest_season_rollover", "换季提醒读取失败")
		} else {
			result.LatestSeasonRollover = notificationInfoPayload(notification)
			result.Availability["latest_season_rollover"] = true
		}
	}

	if l.svcCtx.FriendModel == nil {
		appendBootstrapPartial(result, "pending_friend_request_count", "好友服务不可用")
	} else if pendingCount, countErr := l.svcCtx.FriendModel.GetPendingRequestCount(userID); countErr != nil {
		appendBootstrapPartial(result, "pending_friend_request_count", "好友申请数读取失败")
	} else {
		result.PendingFriendRequestCount = int(pendingCount)
		result.Availability["pending_friend_request_count"] = true
	}

	if !l.svcCtx.CompetitiveReadModelsEnabled() {
		appendBootstrapPartial(result, "competitive_revision", "竞技读模型尚未切换")
	} else if l.svcCtx.CompetitiveReadModel == nil {
		appendBootstrapPartial(result, "competitive_revision", "竞技快照服务不可用")
	} else if stats, statsErr := l.svcCtx.CompetitiveReadModel.FindStats(userID, 0); statsErr != nil {
		appendBootstrapPartial(result, "competitive_revision", "竞技版本读取失败")
	} else {
		if stats != nil {
			result.CompetitiveRevision = stats.Revision
		}
		result.Availability["competitive_revision"] = true
	}
	return result, nil
}

func appendBootstrapPartial(resp *types.GetUserBootstrapResp, scope, message string) {
	if resp == nil {
		return
	}
	resp.Availability[scope] = false
	resp.PartialErrors = append(resp.PartialErrors, types.ReadPartialError{Scope: scope, Message: message})
}

func notificationInfoPayload(notification *model.Notification) *types.NotificationInfo {
	if notification == nil {
		return nil
	}
	data := ""
	if notification.Data != nil {
		data = *notification.Data
	}
	return &types.NotificationInfo{
		Id:        notification.Id,
		Type:      notification.Type,
		Title:     notification.Title,
		Content:   notification.Content,
		Data:      data,
		IsRead:    notification.IsRead == 1,
		CreatedAt: notification.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}
