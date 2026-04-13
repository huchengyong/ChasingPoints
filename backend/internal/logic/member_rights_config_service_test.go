package logic

import (
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newMemberRightsConfigServiceTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.MemberRightsConfig{}); err != nil {
		t.Fatalf("prepare member rights config schema: %v", err)
	}
	return &svc.ServiceContext{
		DB:                      db,
		MemberRightsConfigModel: model.NewMemberRightsConfigModel(db),
	}
}

func TestMemberRightsConfigServiceFallsBackToDefaultsWhenMissing(t *testing.T) {
	svcCtx := newMemberRightsConfigServiceTestSvc(t)
	cfg, err := NewMemberRightsConfigService(svcCtx).GetConfig()
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if cfg.GrowthRules.DailyCap != 5 || cfg.RankingRights.DailyCap != 200 {
		t.Fatalf("unexpected default config: %+v", cfg)
	}
}

func TestMemberRightsConfigServiceReturnsPersistedRules(t *testing.T) {
	svcCtx := newMemberRightsConfigServiceTestSvc(t)
	cfg := model.DefaultMemberRightsConfig()
	growth := model.DefaultMemberGrowthRulesConfig()
	growth.DailyCap = 9
	cfg.SetGrowthRules(growth)
	rights := model.DefaultMemberRankingRightsRulesConfig()
	rights.DailyCap = 260
	cfg.SetRankingRightsRules(rights)
	if err := svcCtx.MemberRightsConfigModel.Upsert(cfg); err != nil {
		t.Fatalf("upsert config: %v", err)
	}

	got, err := NewMemberRightsConfigService(svcCtx).GetConfig()
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if got.GrowthRules.DailyCap != 9 || got.RankingRights.DailyCap != 260 {
		t.Fatalf("unexpected persisted config: %+v", got)
	}
}
