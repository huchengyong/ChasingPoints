package achievement

import (
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func TestAchievementListAndUnlockedPayloadIncludeV2Metadata(t *testing.T) {
	svcCtx, db := newHonorWallTestSvc(t)
	seedHonorWallUser(t, svcCtx, 1001, "本人")
	definitions := []model.Achievement{
		{Key: "chasing_golden_break_1", Name: "小金初现", Description: "九球追分小金", Icon: "/static/images/achievements/chasing_golden_break_1.png", Category: "special", GameType: 2, MetricKey: MetricGoldenBreakTotal, ProgressMode: ProgressModeSum, Threshold: 1, RewardTitleName: "小金猎手", Sort: 1, Status: 1},
		{Key: "legacy_disabled", Name: "已停用", Category: "special", GameType: 4, MetricKey: MetricGoldenBreakTotal, ProgressMode: ProgressModeSum, Threshold: 1, Sort: 2, Status: 0},
	}
	if err := db.Create(&definitions).Error; err != nil {
		t.Fatalf("seed definitions: %v", err)
	}
	if err := db.Model(&model.Achievement{}).Where("id = ?", definitions[1].Id).Update("status", 0).Error; err != nil {
		t.Fatalf("disable legacy definition: %v", err)
	}
	unlockedAt := time.Now()
	if err := db.Create(&[]model.UserAchievement{
		{UserId: 1001, AchievementId: definitions[0].Id, Progress: 1, Unlocked: 1, UnlockedAt: &unlockedAt},
		{UserId: 1001, AchievementId: definitions[1].Id, Progress: 1, Unlocked: 1, UnlockedAt: &unlockedAt},
	}).Error; err != nil {
		t.Fatalf("seed user achievements: %v", err)
	}

	listResp, err := NewGetAchievementListLogic(honorWallContext(1001), svcCtx).GetAchievementList(&types.GetAchievementListReq{})
	if err != nil {
		t.Fatalf("get achievement list: %v", err)
	}
	if !listResp.Success || len(listResp.List) != 1 {
		t.Fatalf("achievement list should include only active definitions: %+v", listResp)
	}
	item := listResp.List[0]
	if item.GameType != 2 || item.RewardTitleName != "小金猎手" || !item.Unlocked || item.Progress != 1 {
		t.Fatalf("achievement list missing V2 metadata: %+v", item)
	}

	unlockedResp, err := NewGetUserAchievementsLogic(honorWallContext(1001), svcCtx).GetUserAchievements()
	if err != nil {
		t.Fatalf("get unlocked achievements: %v", err)
	}
	if !unlockedResp.Success || unlockedResp.Total != 1 || len(unlockedResp.List) != 1 {
		t.Fatalf("unlocked list should hide disabled definitions: %+v", unlockedResp)
	}
	if unlockedResp.List[0].GameType != 2 || unlockedResp.List[0].RewardTitleName != "小金猎手" {
		t.Fatalf("unlocked payload missing V2 metadata: %+v", unlockedResp.List[0])
	}
}
