package notification

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetNotificationPreferencesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取通知偏好
func NewGetNotificationPreferencesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetNotificationPreferencesLogic {
	return &GetNotificationPreferencesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetNotificationPreferencesLogic) GetNotificationPreferences() (resp *types.NotificationPreferencesResp, err error) {
	if l == nil || l.svcCtx == nil || l.svcCtx.UserNotificationPreferenceModel == nil {
		return &types.NotificationPreferencesResp{Success: false}, nil
	}

	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.NotificationPreferencesResp{Success: false}, nil
	}

	prefs, err := l.svcCtx.UserNotificationPreferenceModel.GetByUserIdOrDefault(userIdInt)
	if err != nil {
		l.Logger.Errorf("获取通知偏好失败: %v", err)
		return &types.NotificationPreferencesResp{Success: false}, nil
	}

	return buildNotificationPreferencesResp(prefs), nil
}

func buildNotificationPreferencesResp(prefs *model.UserNotificationPreference) *types.NotificationPreferencesResp {
	if prefs == nil {
		prefs = model.DefaultUserNotificationPreference(0)
	}

	return &types.NotificationPreferencesResp{
		Success:              true,
		MatchResultEnabled:   prefs.MatchResultEnabled,
		FriendRequestEnabled: prefs.FriendRequestEnabled,
		ChallengeEnabled:     prefs.ChallengeEnabled,
		TournamentEnabled:    prefs.TournamentEnabled,
		FollowEnabled:        prefs.FollowEnabled,
	}
}
