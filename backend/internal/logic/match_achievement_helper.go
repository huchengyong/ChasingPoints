package logic

import (
	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func normalizeStoredAchievementType(winType string) string {
	if winType == "" || winType == "normal" {
		return ""
	}
	return normalizeAchievementType(winType)
}

func NormalizeStoredAchievementType(winType string) string {
	return normalizeStoredAchievementType(winType)
}

// buildMatchAchievementPayload 汇总单个选手的战绩；历史别名合并到同一字段并累加。
func buildMatchAchievementPayload(list []model.MatchAchievement, actor int) types.MatchAchievement {
	payload := types.MatchAchievement{}
	for _, ach := range list {
		if ach.Actor != actor {
			continue
		}
		switch ach.AchievementType {
		case "break_clear", "break_and_run":
			payload.BreakClear += ach.Count
		case "continue_clear", "run_out":
			payload.ContinueClear += ach.Count
		case "golden_break", "small_gold":
			payload.GoldenBreak += ach.Count
		case "nine_on_break", "big_gold":
			payload.NineOnBreak += ach.Count
		case "break_50":
			payload.Break50 += ach.Count
		case "break_100":
			payload.Break100 += ach.Count
		case "break_147":
			payload.Break147 += ach.Count
		}
	}
	return payload
}

// BuildMatchAchievementsByActor 分别返回选手1、选手2的战绩。
func BuildMatchAchievementsByActor(list []model.MatchAchievement) (types.MatchAchievement, types.MatchAchievement) {
	return buildMatchAchievementPayload(list, 1), buildMatchAchievementPayload(list, 2)
}
