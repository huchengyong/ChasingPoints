package logic

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
	if !isWin || !memberActive || baseAchievementScore <= 0 {
		return 0
	}
	return baseAchievementScore * memberAchievementMultiplierPercent(memberLevel) / 100
}
