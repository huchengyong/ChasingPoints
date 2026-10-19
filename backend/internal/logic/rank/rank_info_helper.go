package rank

import (
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/ranking"
	"chasing_points/internal/types"
)

func buildRankConfigByLevel(configs []model.RankConfig) map[int]model.RankConfig {
	configByLevel := make(map[int]model.RankConfig, len(configs))
	for _, config := range configs {
		configByLevel[config.Level] = config
	}
	return configByLevel
}

// defaultRankingRow 合成没有段位行的用户在 now 时刻的展示状态（纯读，不建行）。
func defaultRankingRow(rules ranking.Rules, userId int64, gameType int, now time.Time) *model.UserRanking {
	defaults := rules.DefaultsAt(now)
	return &model.UserRanking{
		UserId:        userId,
		GameType:      gameType,
		RankScore:     defaults.RankScore,
		RankLevel:     defaults.RankLevel,
		RatingVersion: defaults.RatingVersion,
	}
}

// placementRemaining 返回排位规则 V2 下剩余定级赛场次；旧规则或尚未切换的行为 0。
func placementRemaining(rules ranking.Rules, row *model.UserRanking, now time.Time) int {
	if row == nil || !rules.V2ActiveAt(now) || row.RatingVersion < model.RatingVersionV2 {
		return 0
	}
	return max(ranking.PlacementMatches-row.RatedMatches, 0)
}

func buildRankInfo(row *model.UserRanking, configByLevel map[int]model.RankConfig, rules ranking.Rules, now time.Time) *types.RankInfo {
	if row == nil {
		return nil
	}
	currentConfig, ok := configByLevel[row.RankLevel]
	if !ok {
		return nil
	}

	var nextLevel int
	var nextName string
	var nextScore int
	var progress int
	if row.RankLevel < 6 {
		if nextConfig, exists := configByLevel[row.RankLevel+1]; exists {
			nextLevel = nextConfig.Level
			nextName = nextConfig.Name
			nextScore = nextConfig.MinScore
			scoreToNext := nextScore - currentConfig.MinScore
			if scoreToNext > 0 {
				progress = (row.RankScore - currentConfig.MinScore) * 100 / scoreToNext
				if progress > 100 {
					progress = 100
				}
				if progress < 0 {
					progress = 0
				}
			}
		}
	} else {
		nextLevel = currentConfig.Level
		nextName = currentConfig.Name
		nextScore = currentConfig.MinScore
		progress = 100
	}

	return &types.RankInfo{
		Level: row.RankLevel, Name: currentConfig.Name, Icon: currentConfig.Icon,
		RankScore: row.RankScore, TotalWins: row.TotalWins, TotalLosses: row.TotalLosses,
		MaxStreak: row.MaxStreak, NextLevel: nextLevel, NextName: nextName,
		NextScore: nextScore, Progress: progress,
		PlacementRemaining: placementRemaining(rules, row, now),
	}
}
