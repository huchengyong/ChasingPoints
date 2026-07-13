package logic

import "chasing_points/internal/model"

const (
	memberAchievementDailyCapDefault = 200
)

const DefaultMemberAchievementDailyCap = memberAchievementDailyCapDefault

func memberAchievementMultiplierPercent(level int) int {
	switch {
	case level >= 5:
		return 140
	case level == 4:
		return 130
	case level == 3:
		return 120
	case level == 2:
		return 110
	default:
		return 100
	}
}

func CalculateMemberAchievementRankingScore(baseAchievementScore int, isWin bool, memberActive bool, memberLevel int) int {
	return CalculateMemberAchievementRankingScoreWithPercent(baseAchievementScore, isWin, memberActive, memberAchievementMultiplierPercent(memberLevel))
}

func CalculateMemberAchievementRankingScoreWithPercent(baseAchievementScore int, isWin bool, memberActive bool, multiplierPercent int) int {
	if !isWin || !memberActive || baseAchievementScore <= 0 {
		return 0
	}
	if multiplierPercent <= 0 {
		return 0
	}
	return baseAchievementScore * multiplierPercent / 100
}

func BuildAchievementRewardMapFromRightsRules(rules model.MemberRankingRightsRulesConfig) map[string]int {
	return map[string]int{
		"break_50":      rules.Break50Score,
		"golden_break":  rules.GoldenBreakScore,
		"break_and_run": rules.BreakAndRunScore,
		"run_out":       rules.RunOutScore,
		"break_100":     rules.Break100Score,
		"nine_on_break": rules.NineOnBreakScore,
		"break_147":     rules.Break147Score,
	}
}
