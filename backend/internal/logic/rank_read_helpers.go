package logic

import (
	"fmt"
	"math"

	"chasing_points/internal/model"
	"chasing_points/internal/ranking"
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
	if remark, ok := ranking.ParseRemark(log.Remark); ok {
		return ratingV2Details(remark)
	}

	remark := parseRankSettlementRemark(log.Remark)
	details := []types.RankDetail{
		{Label: "基础分", Value: log.BaseScore},
	}
	if log.AchievementScore != 0 {
		details = append(details, types.RankDetail{Label: "VIP特殊战绩分", Value: log.AchievementScore})
	}
	if remark.MemberAchievementCapAdjustment != 0 {
		details = append(details, types.RankDetail{Label: "VIP特殊战绩每日封顶", Value: remark.MemberAchievementCapAdjustment})
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

// ratingV2Details 按结算顺序展示排位规则 V2 的明细，各项之和等于本场变化。
func ratingV2Details(remark ranking.Remark) []types.RankDetail {
	if !remark.Rated {
		return []types.RankDetail{{Label: "同对手当日第 7 场起不计分", Value: 0}}
	}
	label := "胜负结算"
	if remark.Expected > 0 {
		label = fmt.Sprintf("胜负结算（赛前胜率 %d%%）", int(math.Round(remark.Expected*100)))
	}
	details := []types.RankDetail{{Label: label, Value: remark.Standard}}
	for _, item := range []struct {
		label string
		value int
	}{
		{label: "定级赛加倍", value: remark.Placement},
		{label: "对手定级中减半", value: remark.OpponentDamping},
		{label: "同对手衰减", value: remark.SameOpponent},
		{label: "每日封顶", value: remark.DailyCap},
		{label: "0 分保护", value: remark.Floor},
	} {
		if item.value != 0 {
			details = append(details, types.RankDetail{Label: item.label, Value: item.value})
		}
	}
	return details
}

// RankRulesVersionOfLogs 返回一场对局的排位记录按哪套规则结算：0 未计分，1 旧规则，2 V2。
func RankRulesVersionOfLogs(logs []model.RankChangeLog) int {
	version := 0
	for i := range logs {
		if _, ok := ranking.ParseRemark(logs[i].Remark); ok {
			return ranking.RulesVersionV2
		}
		version = ranking.RulesVersionLegacy
	}
	return version
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
