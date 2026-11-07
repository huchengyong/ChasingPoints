package public

import (
	"crypto/sha256"
	"fmt"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func buildRankConfigItems(configs []model.RankConfig) ([]types.RankConfigItem, map[int]string, string) {
	items := make([]types.RankConfigItem, 0, len(configs))
	names := make(map[int]string, len(configs))
	hash := sha256.New()
	for _, config := range configs {
		items = append(items, types.RankConfigItem{GameType: 0, Level: config.Level, Name: config.Name, Icon: config.Icon, MinScore: config.MinScore})
		names[config.Level] = config.Name
		_, _ = fmt.Fprintf(hash, "%d:%s:%s:%d;", config.Level, config.Name, config.Icon, config.MinScore)
	}
	return items, names, fmt.Sprintf("%x", hash.Sum(nil))[:16]
}

func leaderboardItem(entry model.LeaderboardEntry, rank int, rankNames map[int]string) types.LeaderboardItem {
	winRate := 0
	if entry.TotalWins+entry.TotalLosses > 0 {
		winRate = entry.TotalWins * 100 / (entry.TotalWins + entry.TotalLosses)
	}
	return types.LeaderboardItem{
		Rank:      rank,
		UserId:    entry.UserId,
		Nickname:  entry.Nickname,
		Avatar:    entry.Avatar,
		RankLevel: entry.RankLevel,
		RankName:  rankNames[entry.RankLevel],
		RankScore: entry.RankScore,
		WinRate:   winRate,
	}
}
