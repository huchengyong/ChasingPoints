package model

import (
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newReputationConfigTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&ReputationConfig{}); err != nil {
		t.Fatalf("prepare reputation config schema: %v", err)
	}
	return db
}

func TestReputationConfigTableName(t *testing.T) {
	if got := (ReputationConfig{}).TableName(); got != "reputation_configs" {
		t.Fatalf("expected table name reputation_configs, got %s", got)
	}
}

func TestReputationDefaultConfigUsesConservativeRules(t *testing.T) {
	cfg := DefaultReputationConfig()

	base, err := cfg.BaseRules()
	if err != nil {
		t.Fatalf("decode base rules: %v", err)
	}
	recovery, err := cfg.RecoveryRules()
	if err != nil {
		t.Fatalf("decode recovery rules: %v", err)
	}
	detection, err := cfg.DetectionRules()
	if err != nil {
		t.Fatalf("decode detection rules: %v", err)
	}

	if cfg.ConfigKey != DefaultReputationConfigKey {
		t.Fatalf("unexpected config key: %s", cfg.ConfigKey)
	}
	if base.MaxScore != 100 || base.InitialScore != 100 || base.BanThreshold != 60 || base.BanDurationHours != 24 || base.MinScore != 0 {
		t.Fatalf("unexpected base defaults: %+v", base)
	}
	if !recovery.Enabled || recovery.RecoverPerHour != 1 || recovery.RecoverMaxScore != 100 {
		t.Fatalf("unexpected recovery defaults: %+v", recovery)
	}
	if detection.StackPenaltiesPerMatch {
		t.Fatalf("expected single-penalty-per-match default, got %+v", detection)
	}
	if len(detection.DurationRules) != 4 {
		t.Fatalf("expected 4 duration rules, got %+v", detection.DurationRules)
	}

	expectedDurationRules := map[int]ReputationDurationRule{
		1: {GameType: 1, Enabled: true, MinMinutesPerRound: 15, MinTotalRounds: 3, MinTotalDurationMinutes: 45, PenaltyScore: 12},
		2: {GameType: 2, Enabled: true, MinMinutesPerRound: 3, MinTotalRounds: 5, MinTotalDurationMinutes: 18, PenaltyScore: 8},
		3: {GameType: 3, Enabled: true, MinMinutesPerRound: 2, MinTotalRounds: 5, MinTotalDurationMinutes: 15, PenaltyScore: 10},
		4: {GameType: 4, Enabled: true, MinMinutesPerRound: 4, MinTotalRounds: 5, MinTotalDurationMinutes: 20, PenaltyScore: 10},
	}
	for _, rule := range detection.DurationRules {
		expected, ok := expectedDurationRules[rule.GameType]
		if !ok {
			t.Fatalf("unexpected game type rule: %+v", rule)
		}
		if rule != expected {
			t.Fatalf("unexpected duration rule for game type %d: %+v", rule.GameType, rule)
		}
	}

	if !detection.SameOpponentRule.Enabled || detection.SameOpponentRule.WindowMinutes != 30 || detection.SameOpponentRule.MaxMatches != 4 || detection.SameOpponentRule.PenaltyScore != 8 || !detection.SameOpponentRule.RequireSameGameType {
		t.Fatalf("unexpected same opponent rule: %+v", detection.SameOpponentRule)
	}
}

func TestReputationConfigModelRoundTrip(t *testing.T) {
	db := newReputationConfigTestDB(t)
	model := NewReputationConfigModel(db)

	cfg := DefaultReputationConfig()
	cfg.UpdatedBy = 9001
	base := DefaultReputationBaseRules()
	base.BanThreshold = 55
	cfg.SetBaseRules(base)
	recovery := DefaultReputationRecoveryRules()
	recovery.RecoverPerHour = 2
	cfg.SetRecoveryRules(recovery)
	detection := DefaultReputationDetectionRules()
	detection.SameOpponentRule.MaxMatches = 5
	cfg.SetDetectionRules(detection)

	if err := model.Upsert(cfg); err != nil {
		t.Fatalf("upsert config: %v", err)
	}

	stored, err := model.FindByKey(DefaultReputationConfigKey)
	if err != nil {
		t.Fatalf("find config: %v", err)
	}
	if stored == nil {
		t.Fatal("expected stored config")
	}

	storedBase, err := stored.BaseRules()
	if err != nil {
		t.Fatalf("decode stored base rules: %v", err)
	}
	storedRecovery, err := stored.RecoveryRules()
	if err != nil {
		t.Fatalf("decode stored recovery rules: %v", err)
	}
	storedDetection, err := stored.DetectionRules()
	if err != nil {
		t.Fatalf("decode stored detection rules: %v", err)
	}

	if storedBase.BanThreshold != 55 || storedRecovery.RecoverPerHour != 2 || storedDetection.SameOpponentRule.MaxMatches != 5 || stored.UpdatedBy != 9001 {
		t.Fatalf("unexpected stored config: base=%+v recovery=%+v detection=%+v cfg=%+v", storedBase, storedRecovery, storedDetection, stored)
	}
}

func TestReputationConfigModelReturnsHelpfulNilErrors(t *testing.T) {
	var cfgModel *ReputationConfigModel
	if _, err := cfgModel.FindByKey(DefaultReputationConfigKey); !errors.Is(err, ErrReputationConfigDBNil) {
		t.Fatalf("expected nil db error, got %v", err)
	}
}
