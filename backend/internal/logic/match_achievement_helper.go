package logic

import (
	"billiard_master/internal/model"
	"billiard_master/internal/types"
)

func normalizeStoredAchievementType(winType string) string {
	if winType == "" || winType == "normal" {
		return ""
	}
	return normalizeAchievementType(winType)
}

func buildMatchAchievementPayload(list []model.MatchAchievement) types.MatchAchievement {
	payload := types.MatchAchievement{}
	for _, ach := range list {
		switch ach.AchievementType {
		case "break_clear", "break_and_run":
			payload.BreakClear = ach.Count
		case "continue_clear", "run_out":
			payload.ContinueClear = ach.Count
		case "golden_break", "small_gold":
			payload.GoldenBreak = ach.Count
		case "nine_on_break", "big_gold":
			payload.NineOnBreak = ach.Count
		case "break_50":
			payload.Break50 = ach.Count
		case "break_100":
			payload.Break100 = ach.Count
		case "break_147":
			payload.Break147 = ach.Count
		}
	}
	return payload
}
