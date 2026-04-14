package model

import (
	"errors"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newUserReputationProfileTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&UserReputationProfile{}); err != nil {
		t.Fatalf("prepare user reputation profile schema: %v", err)
	}
	return db
}

func TestReputationProfileTableName(t *testing.T) {
	if got := (UserReputationProfile{}).TableName(); got != "user_reputation_profiles" {
		t.Fatalf("expected table name user_reputation_profiles, got %s", got)
	}
}

func TestReputationProfileModelFindOrCreateAndSave(t *testing.T) {
	db := newUserReputationProfileTestDB(t)
	model := NewUserReputationProfileModel(db)

	profile, err := model.FindOrCreateWithTx(nil, 1001, 87)
	if err != nil {
		t.Fatalf("find or create profile: %v", err)
	}
	if profile.UserID != 1001 || profile.ReputationScore != 87 || profile.TotalPenaltyCount != 0 || profile.TotalAbnormalMatchCount != 0 {
		t.Fatalf("unexpected initial profile: %+v", profile)
	}

	recoveredAt := time.Date(2026, 4, 13, 10, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	penalizedAt := recoveredAt.Add(2 * time.Hour)
	banUntil := penalizedAt.Add(24 * time.Hour)
	profile.ReputationScore = 58
	profile.LastRecoveredAt = &recoveredAt
	profile.LastPenalizedAt = &penalizedAt
	profile.BanUntil = &banUntil
	profile.TotalPenaltyCount = 2
	profile.TotalAbnormalMatchCount = 3
	if err := model.Save(profile); err != nil {
		t.Fatalf("save profile: %v", err)
	}

	reloaded, err := model.FindByUserID(1001)
	if err != nil {
		t.Fatalf("reload profile: %v", err)
	}
	if reloaded == nil {
		t.Fatal("expected reloaded profile")
	}
	if reloaded.ReputationScore != 58 || reloaded.TotalPenaltyCount != 2 || reloaded.TotalAbnormalMatchCount != 3 {
		t.Fatalf("unexpected reloaded profile counters: %+v", reloaded)
	}
	if reloaded.LastRecoveredAt == nil || !reloaded.LastRecoveredAt.Equal(recoveredAt) {
		t.Fatalf("unexpected last recovered at: %+v", reloaded.LastRecoveredAt)
	}
	if reloaded.LastPenalizedAt == nil || !reloaded.LastPenalizedAt.Equal(penalizedAt) {
		t.Fatalf("unexpected last penalized at: %+v", reloaded.LastPenalizedAt)
	}
	if reloaded.BanUntil == nil || !reloaded.BanUntil.Equal(banUntil) {
		t.Fatalf("unexpected ban until: %+v", reloaded.BanUntil)
	}
}

func TestReputationProfileModelReturnsHelpfulNilErrors(t *testing.T) {
	var profileModel *UserReputationProfileModel
	if _, err := profileModel.FindByUserID(1); !errors.Is(err, ErrUserReputationProfileDBNil) {
		t.Fatalf("expected nil db error, got %v", err)
	}
	if _, err := profileModel.FindOrCreateWithTx(nil, 1, 88); !errors.Is(err, ErrUserReputationProfileDBNil) {
		t.Fatalf("expected find or create nil db error, got %v", err)
	}
	if err := profileModel.SaveWithTx(nil, &UserReputationProfile{}); !errors.Is(err, ErrUserReputationProfileDBNil) {
		t.Fatalf("expected save nil db error, got %v", err)
	}
}
