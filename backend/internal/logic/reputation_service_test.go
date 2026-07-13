package logic

import (
	"errors"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/gorm"
)

func seedReputationConfig(t *testing.T, svcCtx *svc.ServiceContext, mutate func(*model.ReputationConfig)) {
	t.Helper()

	cfg := model.DefaultReputationConfig()
	if mutate != nil {
		mutate(cfg)
	}
	if err := svcCtx.ReputationConfigModel.Upsert(cfg); err != nil {
		t.Fatalf("upsert reputation config: %v", err)
	}
}

func mustCreateReputationProfile(t *testing.T, svcCtx *svc.ServiceContext, profile *model.UserReputationProfile) {
	t.Helper()
	if err := svcCtx.UserReputationProfileModel.Save(profile); err != nil {
		t.Fatalf("save reputation profile: %v", err)
	}
}

func TestReputationServiceGetOrCreateProfileUsesConfiguredInitialScore(t *testing.T) {
	now := time.Date(2026, 4, 13, 10, 30, 0, 0, UTC8Location)
	svcCtx := newReputationLogicTestSvc(t)
	seedReputationConfig(t, svcCtx, func(cfg *model.ReputationConfig) {
		baseRules := model.DefaultReputationBaseRules()
		baseRules.InitialScore = 88
		cfg.SetBaseRules(baseRules)
	})

	profile, err := NewReputationService(svcCtx, func() time.Time { return now }).GetOrCreateProfile(101)
	if err != nil {
		t.Fatalf("get or create profile: %v", err)
	}

	if profile.ReputationScore != 88 {
		t.Fatalf("expected configured initial score, got %+v", profile)
	}
	if profile.LastRecoveredAt == nil || !profile.LastRecoveredAt.Equal(now) {
		t.Fatalf("expected last recovered at to initialize to now, got %+v", profile)
	}
}

func TestReputationServiceGetOrCreateProfileCapsInitialScoreAtMaxScore(t *testing.T) {
	now := time.Date(2026, 4, 13, 10, 30, 0, 0, UTC8Location)
	svcCtx := newReputationLogicTestSvc(t)
	seedReputationConfig(t, svcCtx, func(cfg *model.ReputationConfig) {
		baseRules := model.DefaultReputationBaseRules()
		baseRules.InitialScore = 120
		baseRules.MaxScore = 90
		cfg.SetBaseRules(baseRules)
	})

	profile, err := NewReputationService(svcCtx, func() time.Time { return now }).GetOrCreateProfile(107)
	if err != nil {
		t.Fatalf("get or create profile: %v", err)
	}

	if profile.ReputationScore != 90 {
		t.Fatalf("expected initial score capped at max score, got %+v", profile)
	}
}

func TestReputationServiceLazyRecoveryUsesWholeHoursOnlyAndUpdatesLastRecoveredAt(t *testing.T) {
	now := time.Date(2026, 4, 13, 10, 30, 0, 0, UTC8Location)
	lastRecoveredAt := now.Add(-(2*time.Hour + 30*time.Minute))
	svcCtx := newReputationLogicTestSvc(t)
	seedReputationConfig(t, svcCtx, func(cfg *model.ReputationConfig) {
		recoveryRules := model.DefaultReputationRecoveryRules()
		recoveryRules.RecoverPerHour = 4
		recoveryRules.RecoverMaxScore = 100
		cfg.SetRecoveryRules(recoveryRules)
	})
	mustCreateReputationProfile(t, svcCtx, &model.UserReputationProfile{
		UserID:          102,
		ReputationScore: 70,
		LastRecoveredAt: &lastRecoveredAt,
	})

	profile, err := NewReputationService(svcCtx, func() time.Time { return now }).GetOrCreateProfile(102)
	if err != nil {
		t.Fatalf("get or create profile: %v", err)
	}

	if profile.ReputationScore != 78 {
		t.Fatalf("expected 2 whole hours of recovery, got %+v", profile)
	}
	wantRecoveredAt := lastRecoveredAt.Add(2 * time.Hour)
	if profile.LastRecoveredAt == nil || !profile.LastRecoveredAt.Equal(wantRecoveredAt) {
		t.Fatalf("expected last recovered at %s, got %+v", wantRecoveredAt, profile)
	}
}

func TestReputationServiceLazyRecoveryDoesNotExceedRecoverMaxScore(t *testing.T) {
	now := time.Date(2026, 4, 13, 10, 30, 0, 0, UTC8Location)
	lastRecoveredAt := now.Add(-3 * time.Hour)
	svcCtx := newReputationLogicTestSvc(t)
	seedReputationConfig(t, svcCtx, func(cfg *model.ReputationConfig) {
		baseRules := model.DefaultReputationBaseRules()
		baseRules.MaxScore = 90
		cfg.SetBaseRules(baseRules)

		recoveryRules := model.DefaultReputationRecoveryRules()
		recoveryRules.RecoverPerHour = 4
		recoveryRules.RecoverMaxScore = 95
		cfg.SetRecoveryRules(recoveryRules)
	})
	mustCreateReputationProfile(t, svcCtx, &model.UserReputationProfile{
		UserID:          103,
		ReputationScore: 89,
		LastRecoveredAt: &lastRecoveredAt,
	})

	profile, err := NewReputationService(svcCtx, func() time.Time { return now }).GetOrCreateProfile(103)
	if err != nil {
		t.Fatalf("get or create profile: %v", err)
	}

	if profile.ReputationScore != 90 {
		t.Fatalf("expected score capped at recover max, got %+v", profile)
	}
}

func TestReputationServiceApplyPenaltyWritesProfileAndLog(t *testing.T) {
	now := time.Date(2026, 4, 13, 10, 30, 0, 0, UTC8Location)
	svcCtx := newReputationLogicTestSvc(t)
	seedReputationConfig(t, svcCtx, nil)

	service := NewReputationService(svcCtx, func() time.Time { return now })
	result, err := service.ApplyPenalty(ReputationPenaltyInput{
		UserID:               104,
		MatchID:              9001,
		PenaltyScore:         12,
		ReasonCode:           model.ReputationReasonDurationAbnormal,
		ReasonDetail:         "duration too short",
		CountAsAbnormalMatch: true,
	})
	if err != nil {
		t.Fatalf("apply penalty: %v", err)
	}

	if !result.Applied || result.Duplicate {
		t.Fatalf("unexpected apply penalty result: %+v", result)
	}
	if result.Profile == nil || result.Profile.ReputationScore != 88 {
		t.Fatalf("unexpected saved profile: %+v", result.Profile)
	}
	if result.Profile.TotalPenaltyCount != 1 || result.Profile.TotalAbnormalMatchCount != 1 {
		t.Fatalf("unexpected counters after penalty: %+v", result.Profile)
	}
	if result.Profile.LastPenalizedAt == nil || !result.Profile.LastPenalizedAt.Equal(now) {
		t.Fatalf("expected last penalized at %s, got %+v", now, result.Profile)
	}
	if result.Log == nil || result.Log.ChangeScore != -12 || result.Log.BeforeScore != 100 || result.Log.AfterScore != 88 {
		t.Fatalf("unexpected penalty log: %+v", result.Log)
	}
}

func TestReputationServiceApplyPenaltyIsIdempotentPerReasonCode(t *testing.T) {
	now := time.Date(2026, 4, 13, 10, 30, 0, 0, UTC8Location)
	svcCtx := newReputationLogicTestSvc(t)
	seedReputationConfig(t, svcCtx, nil)

	service := NewReputationService(svcCtx, func() time.Time { return now })
	input := ReputationPenaltyInput{
		UserID:               105,
		MatchID:              9002,
		PenaltyScore:         10,
		ReasonCode:           model.ReputationReasonDurationAbnormal,
		CountAsAbnormalMatch: true,
	}

	first, err := service.ApplyPenalty(input)
	if err != nil {
		t.Fatalf("first apply penalty: %v", err)
	}
	second, err := service.ApplyPenalty(input)
	if err != nil {
		t.Fatalf("second apply penalty: %v", err)
	}

	if !first.Applied || first.Duplicate {
		t.Fatalf("unexpected first result: %+v", first)
	}
	if second.Applied || !second.Duplicate {
		t.Fatalf("expected duplicate result, got %+v", second)
	}

	profile, err := svcCtx.UserReputationProfileModel.FindByUserID(105)
	if err != nil {
		t.Fatalf("find profile: %v", err)
	}
	if profile == nil || profile.ReputationScore != 90 || profile.TotalPenaltyCount != 1 {
		t.Fatalf("duplicate penalty should not mutate profile twice: %+v", profile)
	}

	log, err := svcCtx.UserReputationLogModel.FindByUserMatchChangeTypeAndReasonCode(
		105,
		9002,
		model.ReputationChangeTypePenalty,
		model.ReputationReasonDurationAbnormal,
	)
	if err != nil {
		t.Fatalf("find penalty log: %v", err)
	}
	if log == nil {
		t.Fatal("expected penalty log to exist")
	}
}

func TestReputationServiceApplyPenaltyExtendsBanFromExistingBanUntil(t *testing.T) {
	now := time.Date(2026, 4, 13, 10, 30, 0, 0, UTC8Location)
	existingBanUntil := now.Add(12 * time.Hour)
	svcCtx := newReputationLogicTestSvc(t)
	seedReputationConfig(t, svcCtx, func(cfg *model.ReputationConfig) {
		baseRules := model.DefaultReputationBaseRules()
		baseRules.InitialScore = 65
		baseRules.BanThreshold = 60
		baseRules.BanDurationHours = 24
		cfg.SetBaseRules(baseRules)
	})
	mustCreateReputationProfile(t, svcCtx, &model.UserReputationProfile{
		UserID:          106,
		ReputationScore: 65,
		BanUntil:        &existingBanUntil,
		LastRecoveredAt: &now,
	})

	result, err := NewReputationService(svcCtx, func() time.Time { return now }).ApplyPenalty(ReputationPenaltyInput{
		UserID:               106,
		MatchID:              9003,
		PenaltyScore:         10,
		ReasonCode:           model.ReputationReasonSameOpponentHighFrequency,
		CountAsAbnormalMatch: true,
	})
	if err != nil {
		t.Fatalf("apply penalty: %v", err)
	}

	wantBanUntil := existingBanUntil.Add(24 * time.Hour)
	if result.Profile == nil || result.Profile.BanUntil == nil || !result.Profile.BanUntil.Equal(wantBanUntil) {
		t.Fatalf("expected ban until %s, got %+v", wantBanUntil, result.Profile)
	}
}

func TestReputationServiceApplyPenaltyDoesNotIncrementAbnormalMatchCountWhenDisabled(t *testing.T) {
	now := time.Date(2026, 4, 13, 10, 30, 0, 0, UTC8Location)
	svcCtx := newReputationLogicTestSvc(t)
	seedReputationConfig(t, svcCtx, nil)

	result, err := NewReputationService(svcCtx, func() time.Time { return now }).ApplyPenalty(ReputationPenaltyInput{
		UserID:       108,
		MatchID:      9004,
		PenaltyScore: 6,
		ReasonCode:   model.ReputationReasonDurationAbnormal,
	})
	if err != nil {
		t.Fatalf("apply penalty: %v", err)
	}

	if result.Profile == nil {
		t.Fatal("expected profile after penalty")
	}
	if result.Profile.TotalPenaltyCount != 1 {
		t.Fatalf("expected total penalty count to grow, got %+v", result.Profile)
	}
	if result.Profile.TotalAbnormalMatchCount != 0 {
		t.Fatalf("expected abnormal match count to stay unchanged, got %+v", result.Profile)
	}
}

func TestReputationServiceRollsBackProfileWhenLogCreateFails(t *testing.T) {
	now := time.Date(2026, 4, 13, 10, 30, 0, 0, UTC8Location)
	svcCtx := newReputationLogicTestSvc(t)
	seedReputationConfig(t, svcCtx, nil)

	initialRecoveredAt := now.Add(-time.Hour)
	mustCreateReputationProfile(t, svcCtx, &model.UserReputationProfile{
		UserID:                  109,
		ReputationScore:         80,
		LastRecoveredAt:         &initialRecoveredAt,
		TotalPenaltyCount:       2,
		TotalAbnormalMatchCount: 1,
	})

	callbackName := "reputation_test_fail_log_create"
	if err := svcCtx.DB.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "user_reputation_logs" {
			tx.AddError(errors.New("forced log create failure"))
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	defer svcCtx.DB.Callback().Create().Remove(callbackName)

	_, err := NewReputationService(svcCtx, func() time.Time { return now }).ApplyPenalty(ReputationPenaltyInput{
		UserID:               109,
		MatchID:              9005,
		PenaltyScore:         10,
		ReasonCode:           model.ReputationReasonDurationAbnormal,
		CountAsAbnormalMatch: true,
	})
	if err == nil {
		t.Fatal("expected apply penalty to fail when log create fails")
	}

	profile, err := svcCtx.UserReputationProfileModel.FindByUserID(109)
	if err != nil {
		t.Fatalf("find profile after rollback: %v", err)
	}
	if profile == nil {
		t.Fatal("expected profile to remain after rollback")
	}
	if profile.ReputationScore != 80 || profile.TotalPenaltyCount != 2 || profile.TotalAbnormalMatchCount != 1 {
		t.Fatalf("expected profile changes to roll back, got %+v", profile)
	}

	log, err := svcCtx.UserReputationLogModel.FindByUserMatchChangeTypeAndReasonCode(
		109,
		9005,
		model.ReputationChangeTypePenalty,
		model.ReputationReasonDurationAbnormal,
	)
	if err != nil {
		t.Fatalf("find log after rollback: %v", err)
	}
	if log != nil {
		t.Fatalf("expected no residual log after rollback, got %+v", log)
	}
}

func TestReputationServiceApplyPenaltyReloadsProfileWhenInitialCreateHitsDuplicate(t *testing.T) {
	now := time.Date(2026, 4, 13, 10, 30, 0, 0, UTC8Location)
	svcCtx := newReputationLogicTestSvc(t)
	seedReputationConfig(t, svcCtx, nil)

	callbackName := "reputation_test_profile_duplicate_then_reload"
	injected := false
	if err := svcCtx.DB.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		if injected || tx.Statement == nil || tx.Statement.Table != "user_reputation_profiles" {
			return
		}
		injected = true
		if err := tx.Exec(
			`INSERT INTO user_reputation_profiles
				(user_id, reputation_score, total_penalty_count, total_abnormal_match_count, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			110, 77, 0, 0, now, now,
		).Error; err != nil {
			tx.AddError(err)
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	defer svcCtx.DB.Callback().Create().Remove(callbackName)

	result, err := NewReputationService(svcCtx, func() time.Time { return now }).ApplyPenalty(ReputationPenaltyInput{
		UserID:               110,
		MatchID:              9006,
		PenaltyScore:         5,
		ReasonCode:           model.ReputationReasonDurationAbnormal,
		CountAsAbnormalMatch: true,
	})
	if err != nil {
		t.Fatalf("apply penalty with duplicate profile create: %v", err)
	}

	if !result.Applied || result.Profile == nil {
		t.Fatalf("expected penalty apply after duplicate reload, got %+v", result)
	}
	if result.Profile.ReputationScore != 72 {
		t.Fatalf("expected service to reuse injected profile score, got %+v", result.Profile)
	}
	if result.Log == nil || result.Log.BeforeScore != 77 || result.Log.AfterScore != 72 {
		t.Fatalf("expected log to reflect reloaded profile, got %+v", result.Log)
	}
}
