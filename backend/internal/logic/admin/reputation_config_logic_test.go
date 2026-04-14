package admin

import (
	"context"
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newReputationAdminTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.ReputationConfig{}); err != nil {
		t.Fatalf("prepare reputation config schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:                    db,
		ReputationConfigModel: model.NewReputationConfigModel(db),
	}
}

func newAdminContextWithID(adminID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", -adminID)
}

func TestAdminGetReputationConfigReturnsDefaultsWhenMissing(t *testing.T) {
	svcCtx := newReputationAdminTestSvc(t)

	resp, err := NewAdminGetReputationConfigLogic(context.Background(), svcCtx).AdminGetReputationConfig()
	if err != nil {
		t.Fatalf("get reputation config: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success response, got %#v", resp)
	}
	if resp.BaseRules.MaxScore != 100 || resp.BaseRules.InitialScore != 100 || resp.BaseRules.BanThreshold != 60 {
		t.Fatalf("unexpected default base rules: %#v", resp.BaseRules)
	}
	if resp.RecoveryRules.RecoverPerHour != 1 || resp.RecoveryRules.RecoverMaxScore != 100 {
		t.Fatalf("unexpected default recovery rules: %#v", resp.RecoveryRules)
	}
	if len(resp.DetectionRules.DurationRules) == 0 || resp.DetectionRules.SameOpponentRule.WindowMinutes != 30 {
		t.Fatalf("unexpected default detection rules: %#v", resp.DetectionRules)
	}
}

func TestAdminUpdateReputationConfigRejectsInvalidValues(t *testing.T) {
	svcCtx := newReputationAdminTestSvc(t)
	logic := NewAdminUpdateReputationConfigLogic(newAdminContextWithID(9527), svcCtx)

	testCases := []struct {
		name string
		req  *types.AdminReputationConfigUpdateReq
	}{
		{
			name: "threshold exceeds max score",
			req: &types.AdminReputationConfigUpdateReq{
				BaseRules: types.AdminReputationBaseRules{
					MaxScore:         80,
					InitialScore:     70,
					BanThreshold:     81,
					BanDurationHours: 24,
					MinScore:         0,
				},
				RecoveryRules: types.AdminReputationRecoveryRules{
					Enabled:         true,
					RecoverPerHour:  1,
					RecoverMaxScore: 80,
				},
				DetectionRules: types.AdminReputationDetectionRules{
					DurationRules: []types.AdminReputationDurationRule{
						{GameType: 3, Enabled: true, MinMinutesPerRound: 2, MinTotalRounds: 5, MinTotalDurationMinutes: 15, PenaltyScore: 10},
					},
					SameOpponentRule: types.AdminReputationSameOpponentRule{
						Enabled:             true,
						WindowMinutes:       30,
						MaxMatches:          4,
						PenaltyScore:        8,
						RequireSameGameType: true,
					},
				},
			},
		},
		{
			name: "negative penalty score",
			req: &types.AdminReputationConfigUpdateReq{
				BaseRules: types.AdminReputationBaseRules{
					MaxScore:         100,
					InitialScore:     90,
					BanThreshold:     60,
					BanDurationHours: 24,
					MinScore:         0,
				},
				RecoveryRules: types.AdminReputationRecoveryRules{
					Enabled:         true,
					RecoverPerHour:  1,
					RecoverMaxScore: 100,
				},
				DetectionRules: types.AdminReputationDetectionRules{
					DurationRules: []types.AdminReputationDurationRule{
						{GameType: 3, Enabled: true, MinMinutesPerRound: 2, MinTotalRounds: 5, MinTotalDurationMinutes: 15, PenaltyScore: -1},
					},
					SameOpponentRule: types.AdminReputationSameOpponentRule{
						Enabled:             true,
						WindowMinutes:       30,
						MaxMatches:          4,
						PenaltyScore:        8,
						RequireSameGameType: true,
					},
				},
			},
		},
		{
			name: "initial score below ban threshold",
			req: &types.AdminReputationConfigUpdateReq{
				BaseRules: types.AdminReputationBaseRules{
					MaxScore:         100,
					InitialScore:     59,
					BanThreshold:     60,
					BanDurationHours: 24,
					MinScore:         0,
				},
				RecoveryRules: types.AdminReputationRecoveryRules{
					Enabled:         true,
					RecoverPerHour:  1,
					RecoverMaxScore: 100,
				},
				DetectionRules: types.AdminReputationDetectionRules{
					DurationRules: []types.AdminReputationDurationRule{
						{GameType: 3, Enabled: true, MinMinutesPerRound: 2, MinTotalRounds: 5, MinTotalDurationMinutes: 15, PenaltyScore: 10},
					},
					SameOpponentRule: types.AdminReputationSameOpponentRule{
						Enabled:             true,
						WindowMinutes:       30,
						MaxMatches:          4,
						PenaltyScore:        8,
						RequireSameGameType: true,
					},
				},
			},
		},
		{
			name: "invalid window and rounds",
			req: &types.AdminReputationConfigUpdateReq{
				BaseRules: types.AdminReputationBaseRules{
					MaxScore:         100,
					InitialScore:     90,
					BanThreshold:     60,
					BanDurationHours: 24,
					MinScore:         0,
				},
				RecoveryRules: types.AdminReputationRecoveryRules{
					Enabled:         true,
					RecoverPerHour:  1,
					RecoverMaxScore: 100,
				},
				DetectionRules: types.AdminReputationDetectionRules{
					DurationRules: []types.AdminReputationDurationRule{
						{GameType: 3, Enabled: true, MinMinutesPerRound: 2, MinTotalRounds: 0, MinTotalDurationMinutes: 15, PenaltyScore: 10},
					},
					SameOpponentRule: types.AdminReputationSameOpponentRule{
						Enabled:             true,
						WindowMinutes:       0,
						MaxMatches:          4,
						PenaltyScore:        8,
						RequireSameGameType: true,
					},
				},
			},
		},
		{
			name: "unknown game type",
			req: &types.AdminReputationConfigUpdateReq{
				BaseRules: types.AdminReputationBaseRules{
					MaxScore:         100,
					InitialScore:     90,
					BanThreshold:     60,
					BanDurationHours: 24,
					MinScore:         0,
				},
				RecoveryRules: types.AdminReputationRecoveryRules{
					Enabled:         true,
					RecoverPerHour:  1,
					RecoverMaxScore: 100,
				},
				DetectionRules: types.AdminReputationDetectionRules{
					DurationRules: []types.AdminReputationDurationRule{
						{GameType: 99, Enabled: true, MinMinutesPerRound: 2, MinTotalRounds: 5, MinTotalDurationMinutes: 15, PenaltyScore: 10},
					},
					SameOpponentRule: types.AdminReputationSameOpponentRule{
						Enabled:             true,
						WindowMinutes:       30,
						MaxMatches:          4,
						PenaltyScore:        8,
						RequireSameGameType: true,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := logic.AdminUpdateReputationConfig(tc.req)
			if err != nil {
				t.Fatalf("update reputation config: %v", err)
			}
			if resp.Success || resp.Code != 400 {
				t.Fatalf("expected validation failure, got %#v", resp)
			}
		})
	}
}

func TestAdminReputationConfigRoundTrip(t *testing.T) {
	svcCtx := newReputationAdminTestSvc(t)

	updateResp, err := NewAdminUpdateReputationConfigLogic(newAdminContextWithID(42), svcCtx).AdminUpdateReputationConfig(&types.AdminReputationConfigUpdateReq{
		BaseRules: types.AdminReputationBaseRules{
			MaxScore:         120,
			InitialScore:     95,
			BanThreshold:     55,
			BanDurationHours: 36,
			MinScore:         10,
		},
		RecoveryRules: types.AdminReputationRecoveryRules{
			Enabled:         true,
			RecoverPerHour:  3,
			RecoverMaxScore: 110,
		},
		DetectionRules: types.AdminReputationDetectionRules{
			DurationRules: []types.AdminReputationDurationRule{
				{GameType: 1, Enabled: true, MinMinutesPerRound: 15, MinTotalRounds: 3, MinTotalDurationMinutes: 50, PenaltyScore: 12},
				{GameType: 3, Enabled: true, MinMinutesPerRound: 2, MinTotalRounds: 5, MinTotalDurationMinutes: 18, PenaltyScore: 11},
			},
			SameOpponentRule: types.AdminReputationSameOpponentRule{
				Enabled:             true,
				WindowMinutes:       45,
				MaxMatches:          5,
				PenaltyScore:        9,
				RequireSameGameType: true,
			},
			StackPenaltiesPerMatch: true,
		},
	})
	if err != nil {
		t.Fatalf("update reputation config: %v", err)
	}
	if !updateResp.Success {
		t.Fatalf("expected update success, got %#v", updateResp)
	}

	getResp, err := NewAdminGetReputationConfigLogic(context.Background(), svcCtx).AdminGetReputationConfig()
	if err != nil {
		t.Fatalf("get reputation config: %v", err)
	}
	if !getResp.Success {
		t.Fatalf("expected get success, got %#v", getResp)
	}
	if getResp.BaseRules.MaxScore != 120 || getResp.BaseRules.InitialScore != 95 || getResp.BaseRules.BanThreshold != 55 {
		t.Fatalf("unexpected base rules roundtrip: %#v", getResp.BaseRules)
	}
	if getResp.RecoveryRules.RecoverPerHour != 3 || getResp.RecoveryRules.RecoverMaxScore != 110 {
		t.Fatalf("unexpected recovery rules roundtrip: %#v", getResp.RecoveryRules)
	}
	if len(getResp.DetectionRules.DurationRules) != 2 || !getResp.DetectionRules.StackPenaltiesPerMatch {
		t.Fatalf("unexpected detection rules roundtrip: %#v", getResp.DetectionRules)
	}
	if getResp.DetectionRules.SameOpponentRule.WindowMinutes != 45 || getResp.DetectionRules.SameOpponentRule.MaxMatches != 5 {
		t.Fatalf("unexpected same opponent rule roundtrip: %#v", getResp.DetectionRules.SameOpponentRule)
	}
}
