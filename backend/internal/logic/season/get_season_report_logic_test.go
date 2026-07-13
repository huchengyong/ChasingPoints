package season

import (
	"testing"
	"time"

	"chasing_points/internal/model"
)

func TestBuildSeasonTopAchievementsPreservesUnlockOrder(t *testing.T) {
	now := time.Date(2026, 3, 13, 12, 0, 0, 0, time.UTC)
	later := now.Add(2 * time.Hour)

	unlockedAchievements := []model.UserAchievement{
		{
			AchievementId: 2,
			Progress:      8,
			Unlocked:      1,
			UnlockedAt:    &later,
		},
		{
			AchievementId: 1,
			Progress:      3,
			Unlocked:      1,
			UnlockedAt:    &now,
		},
	}

	achievementDefs := []model.Achievement{
		{
			Id:          1,
			Key:         "golden_break",
			Name:        "黄金开球",
			Description: "开球直接清台",
			Category:    "match",
			Threshold:   1,
		},
		{
			Id:          2,
			Key:         "run_out",
			Name:        "连续清台",
			Description: "连续完成清台",
			Category:    "match",
			Threshold:   5,
		},
	}

	list := buildSeasonTopAchievements(unlockedAchievements, achievementDefs)
	if len(list) != 2 {
		t.Fatalf("expected 2 achievements, got %d", len(list))
	}
	if list[0].Id != 2 || list[0].Name != "连续清台" {
		t.Fatalf("expected latest unlocked achievement first, got %+v", list[0])
	}
	if list[1].Id != 1 || list[1].UnlockedAt == "" {
		t.Fatalf("expected second achievement with unlocked time, got %+v", list[1])
	}
}

func TestBuildSeasonTopAchievementsSkipsMissingDefs(t *testing.T) {
	unlockedAchievements := []model.UserAchievement{
		{
			AchievementId: 99,
			Progress:      1,
			Unlocked:      1,
		},
	}

	list := buildSeasonTopAchievements(unlockedAchievements, []model.Achievement{})
	if len(list) != 0 {
		t.Fatalf("expected empty list when defs are missing, got %d", len(list))
	}
}
