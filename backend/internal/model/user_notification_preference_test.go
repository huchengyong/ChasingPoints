package model

import (
	"testing"

	"chasing_points/internal/testsupport"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newUserNotificationPreferenceTestModel(t *testing.T) *UserNotificationPreferenceModel {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := testsupport.PrepareNotificationSchema(db); err != nil {
		t.Fatalf("prepare notification schema: %v", err)
	}

	return NewUserNotificationPreferenceModel(db)
}

func TestUserNotificationPreferenceGetByUserIdOrDefaultReturnsAllEnabledDefaults(t *testing.T) {
	model := newUserNotificationPreferenceTestModel(t)

	prefs, err := model.GetByUserIdOrDefault(1001)
	if err != nil {
		t.Fatalf("get preferences by default: %v", err)
	}
	if prefs == nil {
		t.Fatal("expected default preferences")
	}
	if !prefs.MatchResultEnabled || !prefs.FriendRequestEnabled || !prefs.ChallengeEnabled || !prefs.TournamentEnabled || !prefs.FollowEnabled {
		t.Fatalf("expected all switches enabled by default, got %#v", prefs)
	}
}

func TestUserNotificationPreferenceUpsertCreatesAndUpdatesRow(t *testing.T) {
	model := newUserNotificationPreferenceTestModel(t)

	if err := model.Upsert(&UserNotificationPreference{
		UserId:               1002,
		MatchResultEnabled:   false,
		FriendRequestEnabled: true,
		ChallengeEnabled:     false,
		TournamentEnabled:    true,
		FollowEnabled:        false,
	}); err != nil {
		t.Fatalf("create preferences: %v", err)
	}

	prefs, err := model.FindByUserId(1002)
	if err != nil {
		t.Fatalf("find created preferences: %v", err)
	}
	if prefs == nil {
		t.Fatal("expected persisted preferences")
	}
	if prefs.MatchResultEnabled || prefs.ChallengeEnabled || prefs.FollowEnabled {
		t.Fatalf("expected disabled switches persisted, got %#v", prefs)
	}

	if err := model.Upsert(&UserNotificationPreference{
		UserId:               1002,
		MatchResultEnabled:   true,
		FriendRequestEnabled: false,
		ChallengeEnabled:     true,
		TournamentEnabled:    false,
		FollowEnabled:        true,
	}); err != nil {
		t.Fatalf("update preferences: %v", err)
	}

	updated, err := model.FindByUserId(1002)
	if err != nil {
		t.Fatalf("find updated preferences: %v", err)
	}
	if updated == nil {
		t.Fatal("expected updated preferences")
	}
	if !updated.MatchResultEnabled || updated.FriendRequestEnabled || !updated.ChallengeEnabled || updated.TournamentEnabled || !updated.FollowEnabled {
		t.Fatalf("expected updated switches persisted, got %#v", updated)
	}
}
