package logic

import (
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newReputationLogicTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.ReputationConfig{}, &model.UserReputationProfile{}, &model.UserReputationLog{}); err != nil {
		t.Fatalf("prepare reputation logic schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:                         db,
		ReputationConfigModel:      model.NewReputationConfigModel(db),
		UserReputationProfileModel: model.NewUserReputationProfileModel(db),
		UserReputationLogModel:     model.NewUserReputationLogModel(db),
	}
}

func TestReputationConfigServiceFallsBackToDefaultsWhenMissing(t *testing.T) {
	svcCtx := newReputationLogicTestSvc(t)

	cfg, err := NewReputationConfigService(svcCtx).GetConfig()
	if err != nil {
		t.Fatalf("get config: %v", err)
	}

	if cfg.ConfigKey != model.DefaultReputationConfigKey {
		t.Fatalf("unexpected config key: %+v", cfg)
	}
	if cfg.BaseRules.InitialScore != 100 || cfg.BaseRules.BanThreshold != 60 || cfg.RecoveryRules.RecoverPerHour != 1 {
		t.Fatalf("unexpected fallback config: %+v", cfg)
	}
	if cfg.DetectionRules.SameOpponentRule.WindowMinutes != 30 || len(cfg.DetectionRules.DurationRules) != 4 {
		t.Fatalf("unexpected fallback detection rules: %+v", cfg.DetectionRules)
	}
}

func TestReputationConfigServiceReturnsPersistedRules(t *testing.T) {
	svcCtx := newReputationLogicTestSvc(t)

	cfg := model.DefaultReputationConfig()
	baseRules := model.DefaultReputationBaseRules()
	baseRules.InitialScore = 88
	baseRules.BanThreshold = 50
	cfg.SetBaseRules(baseRules)

	recoveryRules := model.DefaultReputationRecoveryRules()
	recoveryRules.RecoverPerHour = 3
	recoveryRules.RecoverMaxScore = 92
	cfg.SetRecoveryRules(recoveryRules)

	detectionRules := model.DefaultReputationDetectionRules()
	detectionRules.StackPenaltiesPerMatch = true
	detectionRules.SameOpponentRule.MaxMatches = 6
	cfg.SetDetectionRules(detectionRules)

	if err := svcCtx.ReputationConfigModel.Upsert(cfg); err != nil {
		t.Fatalf("upsert config: %v", err)
	}

	got, err := NewReputationConfigService(svcCtx).GetConfig()
	if err != nil {
		t.Fatalf("get config: %v", err)
	}

	if got.BaseRules.InitialScore != 88 || got.BaseRules.BanThreshold != 50 {
		t.Fatalf("unexpected base rules: %+v", got.BaseRules)
	}
	if got.RecoveryRules.RecoverPerHour != 3 || got.RecoveryRules.RecoverMaxScore != 92 {
		t.Fatalf("unexpected recovery rules: %+v", got.RecoveryRules)
	}
	if !got.DetectionRules.StackPenaltiesPerMatch || got.DetectionRules.SameOpponentRule.MaxMatches != 6 {
		t.Fatalf("unexpected detection rules: %+v", got.DetectionRules)
	}
}
