package logic

import (
	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func calculateAchievementScoreFromItems(items []types.AchievementItem, rewardMap map[string]int) int {
	total := 0
	for _, item := range items {
		if item.Count <= 0 {
			continue
		}
		reward := rewardMap[normalizeAchievementType(item.Type)]
		if reward <= 0 {
			continue
		}
		total += reward * item.Count
	}
	return total
}

func calculateAchievementScoreFromStoredAchievements(items []model.MatchAchievement, rewardMap map[string]int) int {
	total := 0
	for _, item := range items {
		if item.Count <= 0 {
			continue
		}
		reward := rewardMap[normalizeAchievementType(item.AchievementType)]
		if reward <= 0 {
			continue
		}
		total += reward * item.Count
	}
	return total
}

func resolveReplayAchievementScores(
	gameType int,
	rounds []model.MatchRound,
	actions []model.MatchAction,
	storedAchievements []model.MatchAchievement,
	rewardMap map[string]int,
) (int, int) {
	if gameType == 1 && len(actions) > 0 {
		if len(rounds) > 0 {
			return calculateSnookerAchievementScoresByActor(filterSnookerActionsToCompletedRounds(actions, rounds), rewardMap)
		}
		return calculateSnookerAchievementScoresByActor(actions, rewardMap)
	}
	if len(rounds) > 0 {
		return calculateAchievementScoresByActor(rounds, rewardMap)
	}
	return calculateAchievementScoreFromStoredAchievements(storedAchievements, rewardMap), 0
}

func ResolveReplayAchievementScores(
	gameType int,
	rounds []model.MatchRound,
	actions []model.MatchAction,
	storedAchievements []model.MatchAchievement,
	rewardMap map[string]int,
) (int, int) {
	return resolveReplayAchievementScores(gameType, rounds, actions, storedAchievements, rewardMap)
}

func applyHistoricalMatchSeasonSnapshot(recordInfo *types.SeasonRecordInfo, season *model.Season, startScore, endScore, peakScore int, hasRankLogs bool) *types.SeasonRecordInfo {
	if recordInfo == nil {
		return &types.SeasonRecordInfo{
			SeasonId:       season.Id,
			SeasonName:     season.Name,
			StartRankScore: startScore,
			EndRankScore:   endScore,
			PeakRankScore:  peakScore,
		}
	}

	if hasRankLogs {
		recordInfo.StartRankScore = startScore
		recordInfo.EndRankScore = endScore
		recordInfo.PeakRankScore = peakScore
	}

	return recordInfo
}

func ApplyHistoricalMatchSeasonSnapshot(recordInfo *types.SeasonRecordInfo, season *model.Season, startScore, endScore, peakScore int, hasRankLogs bool) *types.SeasonRecordInfo {
	return applyHistoricalMatchSeasonSnapshot(recordInfo, season, startScore, endScore, peakScore, hasRankLogs)
}
