package logic

import (
	"chasing_points/internal/model"
	"chasing_points/internal/pkg/ws"
	"chasing_points/internal/svc"
)

type NotificationDispatchInput struct {
	UserId      int64
	Type        string
	Title       string
	Content     string
	Data        *string
	PushTitle   string
	PushContent string
	PushData    map[string]interface{}
	WSCategory  string
}

type NotificationDispatchService struct {
	svcCtx     *svc.ServiceContext
	pushSender func(pushClientId, title, content string, data map[string]interface{})
	wsSender   func(userId int64, category string)
}

func NewNotificationDispatchService(svcCtx *svc.ServiceContext) *NotificationDispatchService {
	service := &NotificationDispatchService{
		svcCtx: svcCtx,
	}
	service.pushSender = service.defaultPushSender
	service.wsSender = service.defaultWSSender
	return service
}

func DispatchNotification(svcCtx *svc.ServiceContext, input NotificationDispatchInput) error {
	return NewNotificationDispatchService(svcCtx).Dispatch(input)
}

func (s *NotificationDispatchService) Dispatch(input NotificationDispatchInput) error {
	if s == nil || s.svcCtx == nil || s.svcCtx.NotificationModel == nil || input.UserId <= 0 || input.Type == "" {
		return nil
	}

	prefs, err := s.loadPreferences(input.UserId)
	if err != nil {
		return err
	}
	if !s.isNotificationEnabled(input.Type, prefs) {
		return nil
	}

	if err := s.svcCtx.NotificationModel.Create(&model.Notification{
		UserId:  input.UserId,
		Type:    input.Type,
		Title:   input.Title,
		Content: input.Content,
		Data:    input.Data,
		IsRead:  0,
	}); err != nil {
		return err
	}

	if s.pushSender != nil && s.svcCtx.UserModel != nil {
		if user, err := s.svcCtx.UserModel.FindById(input.UserId); err == nil && user != nil && user.PushToken != "" {
			pushTitle := input.PushTitle
			if pushTitle == "" {
				pushTitle = input.Title
			}
			pushContent := input.PushContent
			if pushContent == "" {
				pushContent = input.Content
			}
			s.pushSender(user.PushToken, pushTitle, pushContent, input.PushData)
		}
	}

	if s.wsSender != nil {
		category := input.WSCategory
		if category == "" {
			category = input.Type
		}
		s.wsSender(input.UserId, category)
	}

	return nil
}

func (s *NotificationDispatchService) loadPreferences(userId int64) (*model.UserNotificationPreference, error) {
	if s == nil || s.svcCtx == nil || s.svcCtx.UserNotificationPreferenceModel == nil {
		return model.DefaultUserNotificationPreference(userId), nil
	}
	return s.svcCtx.UserNotificationPreferenceModel.GetByUserIdOrDefault(userId)
}

func (s *NotificationDispatchService) isNotificationEnabled(notificationType string, prefs *model.UserNotificationPreference) bool {
	if prefs == nil {
		return true
	}

	switch notificationType {
	case "match_result":
		return prefs.MatchResultEnabled
	case "friend_request":
		return prefs.FriendRequestEnabled
	case "challenge":
		return prefs.ChallengeEnabled
	case "tournament":
		return prefs.TournamentEnabled
	case "follow":
		return prefs.FollowEnabled
	default:
		return true
	}
}

func (s *NotificationDispatchService) defaultPushSender(pushClientId, title, content string, data map[string]interface{}) {
	if s == nil || s.svcCtx == nil || s.svcCtx.PushService == nil {
		return
	}
	s.svcCtx.PushService.SendPush(pushClientId, title, content, data)
}

func (s *NotificationDispatchService) defaultWSSender(userId int64, category string) {
	if ws.GlobalHub == nil {
		return
	}
	ws.GlobalHub.SendToUser(userId, &ws.Message{
		Type: "notification_update",
		Data: map[string]interface{}{
			"category": category,
		},
	})
}
