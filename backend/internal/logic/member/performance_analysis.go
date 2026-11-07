package member

import (
	"math"
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/ranking"
	"chasing_points/internal/types"
)

const (
	performanceSummaryDays       = 90
	performanceSummaryMaxSamples = 200
	performanceMonthCount        = 6
	performanceMonthLayout       = "2006-01"
)

// performanceCandidate 是一条 V2 已计分的本人结算记录，对手赛前分还没有补上。
type performanceCandidate struct {
	matchID    int64
	won        bool
	expected   float64
	raceLength int
	playedAt   time.Time
}

// performanceSample 是可以参与分析的一场排位：本人与对手的结算时数据都齐全。
type performanceSample struct {
	performanceCandidate
	opponentUserID int64
	opponentRating int
}

func (s performanceSample) rankingSample() ranking.PerformanceSample {
	return ranking.PerformanceSample{OpponentRating: s.opponentRating, RaceLength: s.raceLength, Won: s.won}
}

// parsePerformanceRows 解析本人结算记录（最新在前）。
// 只有 V2 记录才有意义：旧规则的分数口径不同，不进入样本，也不进入月末排位分。
// 同对手当日第 7 场起不计分（rated=false）的 V2 记录不进入样本，但分数仍然有效，可作为月末排位分。
func parsePerformanceRows(rows []model.RankChangeLog) (candidates []performanceCandidate, monthEnd map[string]int) {
	monthEnd = make(map[string]int, performanceMonthCount)
	for _, row := range rows {
		remark, ok := ranking.ParseRemark(row.Remark)
		if !ok {
			continue
		}
		playedAt := logicx.InUTC8(row.EffectiveAt)
		if key := playedAt.Format(performanceMonthLayout); !hasKey(monthEnd, key) {
			monthEnd[key] = row.AfterScore
		}
		if !remark.Rated || remark.RaceLength < 1 || (row.Result != "win" && row.Result != "lose") {
			continue
		}
		candidates = append(candidates, performanceCandidate{
			matchID:    row.MatchId,
			won:        row.Result == "win",
			expected:   remark.Expected,
			raceLength: remark.RaceLength,
			playedAt:   playedAt,
		})
	}
	return candidates, monthEnd
}

func hasKey(m map[string]int, key string) bool {
	_, ok := m[key]
	return ok
}

// attachOpponents 补上同场对手的赛前分；对手没有结算记录的样本剔除，返回被剔除的数量。
func attachOpponents(candidates []performanceCandidate, refs []model.OpponentRatingRef) (samples []performanceSample, dropped int) {
	byMatch := make(map[int64]model.OpponentRatingRef, len(refs))
	for _, ref := range refs {
		byMatch[ref.MatchId] = ref
	}
	for _, candidate := range candidates {
		ref, ok := byMatch[candidate.matchID]
		if !ok {
			dropped++
			continue
		}
		samples = append(samples, performanceSample{performanceCandidate: candidate, opponentUserID: ref.UserId, opponentRating: ref.BeforeScore})
	}
	return samples, dropped
}

// summaryWindow 返回 90 天窗口内最近的样本，最多 200 场。samples 必须按时间从新到旧排列。
func summaryWindow(samples []performanceSample, now time.Time) []performanceSample {
	cutoff := now.AddDate(0, 0, -performanceSummaryDays)
	window := make([]performanceSample, 0, min(len(samples), performanceSummaryMaxSamples))
	for _, sample := range samples {
		if sample.playedAt.Before(cutoff) {
			break
		}
		if len(window) == performanceSummaryMaxSamples {
			break
		}
		window = append(window, sample)
	}
	return window
}

func toRatingPayload(rating int, bound ranking.Bound, ok bool) *types.PerformanceRating {
	if !ok {
		return nil
	}
	return &types.PerformanceRating{Rating: rating, Bound: string(bound)}
}

func round(value float64, places int) float64 {
	factor := math.Pow(10, float64(places))
	return math.Round(value*factor) / factor
}

// summarize 汇总 90 天窗口：实际与期望胜场、表现分、最佳胜利与爆冷失利。
func summarize(window []performanceSample) (summary *types.PerformanceSummary, bestWin, worstLoss *performanceSample) {
	wins, expectedWins := 0, 0.0
	rankingSamples := make([]ranking.PerformanceSample, 0, len(window))
	for index := range window {
		sample := &window[index]
		rankingSamples = append(rankingSamples, sample.rankingSample())
		expectedWins += sample.expected
		if sample.won {
			wins++
			// 样本从新到旧：只在严格更小时替换，期望胜率相同时保留较近的一场。
			if bestWin == nil || sample.expected < bestWin.expected {
				bestWin = sample
			}
		} else if worstLoss == nil || sample.expected > worstLoss.expected {
			worstLoss = sample
		}
	}
	rating, bound, ok := ranking.PerformanceRating(rankingSamples, ranking.MinPerformanceSamples)
	return &types.PerformanceSummary{
		Wins:         wins,
		ExpectedWins: round(expectedWins, 2),
		WinsDiff:     round(float64(wins)-expectedWins, 2),
		Rating:       toRatingPayload(rating, bound, ok),
	}, bestWin, worstLoss
}

// monthStart 返回业务时区下 t 所在自然月的月初。
func monthStart(t time.Time) time.Time {
	local := logicx.InUTC8(t)
	return time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, logicx.UTC8Location)
}

// sixMonthsStart 返回实力曲线的起点：当前自然月往前数第 6 个月的月初。
func sixMonthsStart(now time.Time) time.Time {
	return monthStart(now).AddDate(0, -(performanceMonthCount - 1), 0)
}

// buildMonths 生成最近 6 个自然月的实力曲线（从早到晚）。
// 每月样本少于 3 场时表现分为空；读取上限被触发时，被截断的月份（最早读到的记录所在月及更早的月份）
// 标为不完整且表现分为空，避免用残缺样本给出错误的月度值。
func buildMonths(samples []performanceSample, monthEnd map[string]int, now time.Time, truncated bool, oldestAt time.Time) []types.PerformanceMonthItem {
	byMonth := make(map[string][]ranking.PerformanceSample, performanceMonthCount)
	for _, sample := range samples {
		key := sample.playedAt.Format(performanceMonthLayout)
		byMonth[key] = append(byMonth[key], sample.rankingSample())
	}
	var truncatedFrom time.Time
	if truncated && !oldestAt.IsZero() {
		truncatedFrom = monthStart(oldestAt)
	}

	months := make([]types.PerformanceMonthItem, 0, performanceMonthCount)
	for i := performanceMonthCount - 1; i >= 0; i-- {
		start := monthStart(now).AddDate(0, -i, 0)
		key := start.Format(performanceMonthLayout)
		item := types.PerformanceMonthItem{Month: key, SampleCount: len(byMonth[key]), EndRankScore: monthEnd[key]}
		if !truncatedFrom.IsZero() && !start.After(truncatedFrom) {
			item.Incomplete = true
		} else {
			rating, bound, ok := ranking.PerformanceRating(byMonth[key], ranking.MinMonthlyPerformanceSamples)
			item.Rating = toRatingPayload(rating, bound, ok)
		}
		months = append(months, item)
	}
	return months
}

// buildMonthPreview 生成当前自然月的表现分预览，口径与实力曲线的当月值相同。
func buildMonthPreview(samples []performanceSample, now time.Time) *types.PerformanceMonthPreview {
	key := monthStart(now).Format(performanceMonthLayout)
	var monthSamples []ranking.PerformanceSample
	for _, sample := range samples {
		if sample.playedAt.Format(performanceMonthLayout) == key {
			monthSamples = append(monthSamples, sample.rankingSample())
		}
	}
	rating, bound, ok := ranking.PerformanceRating(monthSamples, ranking.MinMonthlyPerformanceSamples)
	return &types.PerformanceMonthPreview{Month: key, SampleCount: len(monthSamples), Rating: toRatingPayload(rating, bound, ok)}
}
