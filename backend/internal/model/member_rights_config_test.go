package model

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newMemberRightsConfigTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&MemberRightsConfig{}); err != nil {
		t.Fatalf("prepare member rights config schema: %v", err)
	}
	return db
}

func TestMemberRightsConfigTableName(t *testing.T) {
	if got := (MemberRightsConfig{}).TableName(); got != "member_rights_configs" {
		t.Fatalf("expected table name member_rights_configs, got %s", got)
	}
}

func TestDefaultMemberRightsConfigUsesAgreedRules(t *testing.T) {
	cfg := DefaultMemberRightsConfig()
	growth, err := cfg.GrowthRules()
	if err != nil {
		t.Fatalf("decode growth rules: %v", err)
	}
	rights, err := cfg.RankingRightsRules()
	if err != nil {
		t.Fatalf("decode ranking rights rules: %v", err)
	}

	if growth.PointsPerCompletedMatch != 1 || growth.DailyCap != 5 {
		t.Fatalf("unexpected growth defaults: %+v", growth)
	}
	if growth.LevelThresholdLv2 != 10 || growth.LevelThresholdLv3 != 60 || growth.LevelThresholdLv4 != 260 || growth.LevelThresholdLv5 != 760 {
		t.Fatalf("unexpected growth thresholds: %+v", growth)
	}
	if growth.ExpireStrategy != "freeze_preserve" {
		t.Fatalf("unexpected expire strategy: %+v", growth)
	}
	if rights.OrdinaryUserAchievementEnabled {
		t.Fatalf("expected ordinary user achievement disabled by default: %+v", rights)
	}
	if rights.DailyCap != 200 || rights.DailyPositiveCap != 500 || rights.Break50Score != 8 || rights.GoldenBreakScore != 4 || rights.BreakAndRunScore != 6 || rights.RunOutScore != 4 || rights.Break100Score != 16 || rights.NineOnBreakScore != 6 || rights.Break147Score != 30 {
		t.Fatalf("unexpected ranking rights score defaults: %+v", rights)
	}
	if rights.Level1Multiplier != 100 || rights.Level2Multiplier != 110 || rights.Level3Multiplier != 120 || rights.Level4Multiplier != 130 || rights.Level5Multiplier != 140 {
		t.Fatalf("unexpected multiplier defaults: %+v", rights)
	}
}

func TestMemberRightsConfigModelRoundTrip(t *testing.T) {
	db := newMemberRightsConfigTestDB(t)
	model := NewMemberRightsConfigModel(db)

	cfg := DefaultMemberRightsConfig()
	cfg.UpdatedBy = 9001
	growth := DefaultMemberGrowthRulesConfig()
	growth.DailyCap = 8
	cfg.SetGrowthRules(growth)
	rights := DefaultMemberRankingRightsRulesConfig()
	rights.DailyCap = 260
	rights.DailyPositiveCap = 520
	cfg.SetRankingRightsRules(rights)

	if err := model.Upsert(cfg); err != nil {
		t.Fatalf("upsert config: %v", err)
	}

	stored, err := model.FindByKey(DefaultMemberRightsConfigKey)
	if err != nil {
		t.Fatalf("find config: %v", err)
	}
	if stored == nil {
		t.Fatal("expected stored config")
	}
	storedGrowth, err := stored.GrowthRules()
	if err != nil {
		t.Fatalf("decode stored growth rules: %v", err)
	}
	storedRights, err := stored.RankingRightsRules()
	if err != nil {
		t.Fatalf("decode stored ranking rules: %v", err)
	}
	if storedGrowth.DailyCap != 8 || storedRights.DailyCap != 260 || storedRights.DailyPositiveCap != 520 || stored.UpdatedBy != 9001 {
		t.Fatalf("unexpected stored config: growth=%+v rights=%+v cfg=%+v", storedGrowth, storedRights, stored)
	}
}
