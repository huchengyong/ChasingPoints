package notification

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type SaveNotificationPreferencesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 保存通知偏好
func NewSaveNotificationPreferencesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SaveNotificationPreferencesLogic {
	return &SaveNotificationPreferencesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SaveNotificationPreferencesLogic) SaveNotificationPreferences(req *types.SaveNotificationPreferencesReq) (resp *types.NotificationPreferencesResp, err error) {
	if l == nil || l.svcCtx == nil || l.svcCtx.UserNotificationPreferenceModel == nil || req == nil {
		return &types.NotificationPreferencesResp{Success: false}, nil
	}

	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.NotificationPreferencesResp{Success: false}, nil
	}

	if err := l.svcCtx.UserNotificationPreferenceModel.Upsert(&model.UserNotificationPreference{
		UserId:               userIdInt,
		MatchResultEnabled:   req.MatchResultEnabled,
		FriendRequestEnabled: req.FriendRequestEnabled,
		ChallengeEnabled:     req.ChallengeEnabled,
		TournamentEnabled:    req.TournamentEnabled,
		FollowEnabled:        req.FollowEnabled,
	}); err != nil {
		l.Logger.Errorf("保存通知偏好失败: %v", err)
		return &types.NotificationPreferencesResp{Success: false}, nil
	}

	prefs, err := l.svcCtx.UserNotificationPreferenceModel.GetByUserIdOrDefault(userIdInt)
	if err != nil {
		l.Logger.Errorf("重新读取通知偏好失败: %v", err)
		return &types.NotificationPreferencesResp{Success: false}, nil
	}

	return buildNotificationPreferencesResp(prefs), nil
}
