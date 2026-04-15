package notification

import (
	"context"
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func notificationLogicCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func newNotificationLogicTestSvc(t *testing.T) *svc.ServiceContext {
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
		UserNotificationPreferenceModel: model.NewUserNotificationPreferenceModel(db),
	}
}

func TestGetNotificationPreferencesDefaultsAllEnabled(t *testing.T) {
	logic := NewGetNotificationPreferencesLogic(notificationLogicCtx(1001), newNotificationLogicTestSvc(t))

	resp, err := logic.GetNotificationPreferences()
	if err != nil {
		t.Fatalf("get notification preferences: %v", err)
	}
	if resp == nil {
		t.Fatal("expected notification preferences response")
	}
	if !resp.Success {
		t.Fatalf("expected success response, got %#v", resp)
	}
	if !resp.MatchResultEnabled || !resp.FriendRequestEnabled || !resp.ChallengeEnabled || !resp.TournamentEnabled || !resp.FollowEnabled {
		t.Fatalf("expected all notification preferences enabled by default, got %#v", resp)
	}
}

func TestSaveNotificationPreferencesPersistsSwitches(t *testing.T) {
	svcCtx := newNotificationLogicTestSvc(t)
	logic := NewSaveNotificationPreferencesLogic(notificationLogicCtx(1002), svcCtx)

	resp, err := logic.SaveNotificationPreferences(&types.SaveNotificationPreferencesReq{
		MatchResultEnabled:   false,
		FriendRequestEnabled: true,
		ChallengeEnabled:     false,
		TournamentEnabled:    true,
		FollowEnabled:        false,
	})
	if err != nil {
		t.Fatalf("save notification preferences: %v", err)
	}
	if resp == nil {
		t.Fatal("expected notification preferences response")
	}
	if !resp.Success {
		t.Fatalf("expected success response, got %#v", resp)
	}
	if resp.MatchResultEnabled || resp.ChallengeEnabled || resp.FollowEnabled {
		t.Fatalf("expected disabled switches to remain false, got %#v", resp)
	}
	if !resp.FriendRequestEnabled || !resp.TournamentEnabled {
		t.Fatalf("expected enabled switches to remain true, got %#v", resp)
	}

	stored, err := svcCtx.UserNotificationPreferenceModel.FindByUserId(1002)
	if err != nil {
		t.Fatalf("reload saved preferences: %v", err)
	}
	if stored == nil {
		t.Fatal("expected saved preferences in db")
	}
	if stored.MatchResultEnabled || stored.ChallengeEnabled || stored.FollowEnabled {
		t.Fatalf("expected disabled switches persisted, got %#v", stored)
	}
}
