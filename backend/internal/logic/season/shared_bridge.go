package season

import (
	logic "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func buildSeasonSnapshotFromLogs(before *model.RankChangeLog, logs []model.RankChangeLog) (start, end, peak int) {
	return logic.BuildSeasonSnapshotFromLogs(before, logs)
}

func buildSeasonRankTrendFromLogs(logs []model.RankChangeLog) []int {
	return logic.BuildSeasonRankTrendFromLogs(logs)
}

func applyHistoricalMatchSeasonSnapshot(recordInfo *types.SeasonRecordInfo, season *model.Season, startScore, endScore, peakScore int, hasRankLogs bool) *types.SeasonRecordInfo {
	return logic.ApplyHistoricalMatchSeasonSnapshot(recordInfo, season, startScore, endScore, peakScore, hasRankLogs)
}
