package rank

import (
	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func buildRankConfigByLevel(configs []model.RankConfig) map[int]model.RankConfig {
	configByLevel := make(map[int]model.RankConfig, len(configs))
	for _, config := range configs {
		configByLevel[config.Level] = config
	}
	return configByLevel
}

func buildRankInfo(ranking *model.UserRanking, configByLevel map[int]model.RankConfig) *types.RankInfo {
	if ranking == nil {
		return nil
	}
	currentConfig, ok := configByLevel[ranking.RankLevel]
	if !ok {
		return nil
	}

	var nextLevel int
	var nextName string
	var nextScore int
	var progress int
	if ranking.RankLevel < 6 {
		if nextConfig, exists := configByLevel[ranking.RankLevel+1]; exists {
			nextLevel = nextConfig.Level
			nextName = nextConfig.Name
			nextScore = nextConfig.MinScore
			scoreToNext := nextScore - currentConfig.MinScore
			if scoreToNext > 0 {
				progress = (ranking.RankScore - currentConfig.MinScore) * 100 / scoreToNext
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
		Level: ranking.RankLevel, Name: currentConfig.Name, Icon: currentConfig.Icon,
		RankScore: ranking.RankScore, TotalWins: ranking.TotalWins, TotalLosses: ranking.TotalLosses,
		MaxStreak: ranking.MaxStreak, NextLevel: nextLevel, NextName: nextName,
		NextScore: nextScore, Progress: progress,
	}
}
