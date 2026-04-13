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

func newMemberRightsAdminTestSvc(t *testing.T) *svc.ServiceContext {
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

func TestAdminGetMemberRightsConfigReturnsDefaultsWhenMissing(t *testing.T) {
	svcCtx := newMemberRightsAdminTestSvc(t)
	resp, err := NewAdminGetMemberRightsConfigLogic(context.Background(), svcCtx).AdminGetMemberRightsConfig()
	if err != nil {
		t.Fatalf("get member rights config: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success response, got %#v", resp)
	}
	if resp.GrowthRules.DailyCap != 5 || resp.RankingRights.DailyCap != 200 {
		t.Fatalf("unexpected default config response: %#v", resp)
	}
}

func TestAdminUpdateMemberRightsConfigRejectsInvalidValues(t *testing.T) {
	svcCtx := newMemberRightsAdminTestSvc(t)
	resp, err := NewAdminUpdateMemberRightsConfigLogic(context.Background(), svcCtx).AdminUpdateMemberRightsConfig(&types.AdminMemberRightsConfigUpdateReq{
		GrowthRules: types.AdminMemberGrowthRules{
			PointsPerCompletedMatch: 0,
			DailyCap:               0,
			LevelThresholdLv2:      10,
			LevelThresholdLv3:      60,
			LevelThresholdLv4:      260,
			LevelThresholdLv5:      760,
			ExpireStrategy:         "freeze_preserve",
		},
		RankingRights: types.AdminMemberRankingRightsRules{
			DailyCap:         200,
			Level1Multiplier: 100,
			Level2Multiplier: 110,
			Level3Multiplier: 120,
			Level4Multiplier: 130,
			Level5Multiplier: 140,
		},
	})
	if err != nil {
		t.Fatalf("update member rights config: %v", err)
	}
	if resp.Success || resp.Code != 400 {
		t.Fatalf("expected validation failure, got %#v", resp)
	}
}

func TestAdminMemberRightsConfigRoundTrip(t *testing.T) {
	svcCtx := newMemberRightsAdminTestSvc(t)
	updateResp, err := NewAdminUpdateMemberRightsConfigLogic(context.Background(), svcCtx).AdminUpdateMemberRightsConfig(&types.AdminMemberRightsConfigUpdateReq{
		GrowthRules: types.AdminMemberGrowthRules{
			PointsPerCompletedMatch: 1,
			DailyCap:               7,
			LevelThresholdLv2:      10,
			LevelThresholdLv3:      60,
			LevelThresholdLv4:      260,
			LevelThresholdLv5:      760,
			ExpireStrategy:         "freeze_preserve",
		},
		RankingRights: types.AdminMemberRankingRightsRules{
			OrdinaryUserAchievementEnabled: false,
			DailyCap:                      260,
			Break50Score:                  8,
			GoldenBreakScore:              4,
			BreakAndRunScore:              6,
			RunOutScore:                   4,
			Break100Score:                 16,
			NineOnBreakScore:              6,
			Break147Score:                 30,
			Level1Multiplier:              100,
			Level2Multiplier:              110,
			Level3Multiplier:              120,
			Level4Multiplier:              130,
			Level5Multiplier:              140,
		},
	})
	if err != nil {
		t.Fatalf("update member rights config: %v", err)
	}
	if !updateResp.Success {
		t.Fatalf("expected update success, got %#v", updateResp)
	}

	getResp, err := NewAdminGetMemberRightsConfigLogic(context.Background(), svcCtx).AdminGetMemberRightsConfig()
	if err != nil {
		t.Fatalf("get member rights config: %v", err)
	}
	if !getResp.Success {
		t.Fatalf("expected get success, got %#v", getResp)
	}
	if getResp.GrowthRules.DailyCap != 7 || getResp.RankingRights.DailyCap != 260 {
		t.Fatalf("unexpected roundtrip config: %#v", getResp)
	}
}
