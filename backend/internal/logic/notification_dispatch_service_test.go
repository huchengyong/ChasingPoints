package logic

import (
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newNotificationDispatchTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := testsupport.PrepareNotificationSchema(db); err != nil {
		t.Fatalf("prepare notification schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:                              db,
		UserModel:                       model.NewUserModel(db),
		NotificationModel:               model.NewNotificationModel(db),
		UserNotificationPreferenceModel: model.NewUserNotificationPreferenceModel(db),
	}
}

func seedNotificationDispatchUser(t *testing.T, svcCtx *svc.ServiceContext, userID int64, pushToken string) {
	t.Helper()

	if err := svcCtx.UserModel.Create(&model.User{
		Id:        userID,
		Nickname:  "测试用户",
		PushToken: pushToken,
		Status:    1,
	}); err != nil {
		t.Fatalf("create dispatch user: %v", err)
	}
}

func TestNotificationDispatchSkipsDisabledType(t *testing.T) {
	svcCtx := newNotificationDispatchTestSvc(t)
	seedNotificationDispatchUser(t, svcCtx, 2001, "push-token")

	if err := svcCtx.UserNotificationPreferenceModel.Upsert(&model.UserNotificationPreference{
		UserId:               2001,
		MatchResultEnabled:   true,
		FriendRequestEnabled: false,
		ChallengeEnabled:     true,
		TournamentEnabled:    true,
		FollowEnabled:        true,
	}); err != nil {
		t.Fatalf("seed disabled friend_request preference: %v", err)
	}

	pushCount := 0
	wsCount := 0
	service := NewNotificationDispatchService(svcCtx)
	service.pushSender = func(pushClientId, title, content string, data map[string]interface{}) {
		pushCount++
	}
	service.wsSender = func(userId int64, category string) {
		wsCount++
	}

	if err := service.Dispatch(NotificationDispatchInput{
		UserId:      2001,
		Type:        "friend_request",
		Title:       "收到好友申请",
		Content:     "有人加你好友",
		PushTitle:   "收到好友申请",
		PushContent: "有人加你好友",
		WSCategory:  "friend_request",
	}); err != nil {
		t.Fatalf("dispatch disabled notification: %v", err)
	}

	var count int64
	if err := svcCtx.DB.Model(&model.Notification{}).Where("user_id = ?", 2001).Count(&count).Error; err != nil {
		t.Fatalf("count notifications: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no notification row when preference disabled, got %d", count)
	}
	if pushCount != 0 || wsCount != 0 {
		t.Fatalf("expected no push/ws when preference disabled, got push=%d ws=%d", pushCount, wsCount)
	}
}

func TestNotificationDispatchCreatesRowAndTriggersPushAndWSWhenEnabled(t *testing.T) {
	svcCtx := newNotificationDispatchTestSvc(t)
	seedNotificationDispatchUser(t, svcCtx, 2002, "push-token")

	pushCount := 0
	wsCount := 0
	service := NewNotificationDispatchService(svcCtx)
	service.pushSender = func(pushClientId, title, content string, data map[string]interface{}) {
		pushCount++
		if pushClientId != "push-token" {
			t.Fatalf("unexpected push token: %s", pushClientId)
		}
	}
	service.wsSender = func(userId int64, category string) {
		wsCount++
		if userId != 2002 || category != "match_result" {
			t.Fatalf("unexpected ws payload user=%d category=%s", userId, category)
		}
	}

	payload := BuildNotificationPayload("/subPages/match/matchResult", 9001, 0)
	if err := service.Dispatch(NotificationDispatchInput{
		UserId:      2002,
		Type:        "match_result",
		Title:       "对局已结束",
		Content:     "结果：胜利（5:3）",
		Data:        payload,
		PushTitle:   "对局已结束",
		PushContent: "结果：胜利（5:3）",
		PushData: map[string]interface{}{
			"url": "/subPages/match/matchResult?match_id=9001",
		},
		WSCategory: "match_result",
	}); err != nil {
		t.Fatalf("dispatch enabled notification: %v", err)
	}

	var notification model.Notification
	if err := svcCtx.DB.Where("user_id = ? AND type = ?", 2002, "match_result").First(&notification).Error; err != nil {
		t.Fatalf("find created notification: %v", err)
	}
	if notification.Title != "对局已结束" || notification.Content != "结果：胜利（5:3）" {
		t.Fatalf("unexpected notification payload: %#v", notification)
	}
	if notification.Data == nil || *notification.Data == "" {
		t.Fatalf("expected notification data payload, got %#v", notification)
	}
	if pushCount != 1 || wsCount != 1 {
		t.Fatalf("expected push/ws exactly once, got push=%d ws=%d", pushCount, wsCount)
	}
}

func TestNotificationDispatchMapsFiveRealBusinessTypes(t *testing.T) {
	service := NewNotificationDispatchService(&svc.ServiceContext{})

	prefs := &model.UserNotificationPreference{
		MatchResultEnabled:   false,
		FriendRequestEnabled: false,
		ChallengeEnabled:     false,
		TournamentEnabled:    false,
		FollowEnabled:        false,
	}

	cases := []struct {
		notificationType string
		expected         bool
	}{
		{notificationType: "match_result", expected: false},
		{notificationType: "friend_request", expected: false},
		{notificationType: "challenge", expected: false},
		{notificationType: "tournament", expected: false},
		{notificationType: "follow", expected: false},
		{notificationType: "unknown_future_type", expected: true},
	}

	for _, tc := range cases {
		got := service.isNotificationEnabled(tc.notificationType, prefs)
		if got != tc.expected {
			t.Fatalf("isNotificationEnabled(%q) = %v, want %v", tc.notificationType, got, tc.expected)
		}
	}

	_ = types.NotificationPreferencesResp{}
}
