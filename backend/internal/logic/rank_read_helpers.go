package logic

import (
	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func buildTrendPointsFromRankChanges(logs []model.RankChangeLog) []types.TrendPoint {
	points := make([]types.TrendPoint, 0, len(logs))
	for _, item := range logs {
		points = append(points, types.TrendPoint{
			MatchId:   item.MatchId,
			Date:      item.EffectiveAt.Format("2006-01-02"),
			RankScore: item.AfterScore,
			Result:    rankResultToTrendResult(item.Result),
		})
	}
	return points
}

func BuildTrendPointsFromRankChanges(logs []model.RankChangeLog) []types.TrendPoint {
	return buildTrendPointsFromRankChanges(logs)
}

func buildSeasonRankTrendFromLogs(logs []model.RankChangeLog) []int {
	trend := make([]int, 0, len(logs))
	for _, item := range logs {
		trend = append(trend, item.AfterScore)
	}
	return trend
}

func BuildSeasonRankTrendFromLogs(logs []model.RankChangeLog) []int {
	return buildSeasonRankTrendFromLogs(logs)
}

func buildSeasonSnapshotFromLogs(before *model.RankChangeLog, logs []model.RankChangeLog) (start, end, peak int) {
	if before != nil {
		start = before.AfterScore
	}

	if len(logs) == 0 {
		return start, start, start
	}

	if before == nil {
		start = logs[0].BeforeScore
	}

	end = logs[len(logs)-1].AfterScore
	peak = start
	for _, item := range logs {
		if item.AfterScore > peak {
			peak = item.AfterScore
		}
	}
	return start, end, peak
}

func BuildSeasonSnapshotFromLogs(before *model.RankChangeLog, logs []model.RankChangeLog) (start, end, peak int) {
	return buildSeasonSnapshotFromLogs(before, logs)
}

func rankResultToTrendResult(result string) int {
	switch result {
	case "win":
		return 1
	case "lose":
		return 2
	default:
		return 3
	}
}

func rankLogToDetails(log *model.RankChangeLog) []types.RankDetail {
	if log == nil {
		return []types.RankDetail{}
	}

	remark := parseRankSettlementRemark(log.Remark)
	details := []types.RankDetail{
		{Label: "基础分", Value: log.BaseScore},
	}
	if log.AchievementScore != 0 {
		details = append(details, types.RankDetail{Label: "会员特殊战绩分", Value: log.AchievementScore})
	}
	if remark.MemberAchievementCapAdjustment != 0 {
		details = append(details, types.RankDetail{Label: "会员特殊战绩每日封顶", Value: remark.MemberAchievementCapAdjustment})
	}
	if remark.LossFloorAdjustment != 0 {
		details = append(details, types.RankDetail{Label: "失败保底", Value: remark.LossFloorAdjustment})
	}
	if remark.SameOpponentAdjustment != 0 {
		label := "同对手衰减"
		if remark.SameOpponentAdjustment > 0 {
			label = "同对手免扣"
		}
		details = append(details, types.RankDetail{Label: label, Value: remark.SameOpponentAdjustment})
	}
	if remark.DailyCapAdjustment != 0 {
		details = append(details, types.RankDetail{Label: "每日封顶", Value: remark.DailyCapAdjustment})
	}
	return details
}

func RankLogToDetails(log *model.RankChangeLog) []types.RankDetail {
	return rankLogToDetails(log)
}

func findOpponentRankLog(logs []model.RankChangeLog, userId int64) *model.RankChangeLog {
	for i := range logs {
		if logs[i].UserId != userId {
			return &logs[i]
		}
	}
	return nil
}

func FindOpponentRankLog(logs []model.RankChangeLog, userId int64) *model.RankChangeLog {
	return findOpponentRankLog(logs, userId)
}
