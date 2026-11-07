package ranking

const (
	// MinPerformanceSamples 是 90 天摘要给出表现分所需的最少样本场数。
	MinPerformanceSamples = 5
	// MinMonthlyPerformanceSamples 是单月给出表现分所需的最少样本场数。
	MinMonthlyPerformanceSamples = 3

	// performanceSearchStep 是上界不够时每次扩展的分数；performanceSearchMaxExpansions 只防止无限循环。
	performanceSearchStep          = 3000
	performanceSearchMaxExpansions = 100
)

// Bound 说明表现分是精确值还是下限、上限：全胜时真实水平可能更高，全负时可能更低。
type Bound string

const (
	BoundExact   Bound = ""
	BoundAtLeast Bound = "at_least"
	BoundAtMost  Bound = "at_most"
)

// PerformanceSample 是一场已计分排位在结算时记录的事实。
type PerformanceSample struct {
	OpponentRating int // 对手赛前排位分
	RaceLength     int // 赛制长度 N
	Won            bool
}

// PerformanceRating 反推一组排位的表现分：求解 R，使 Σ 整场胜率(单局胜率(R, 对手分), N) 等于实际胜场数。
// 公式与结算完全相同，只用于展示，不改变排位分。
//   - 样本少于 minSamples 场时 ok 为 false。
//   - 全胜按「实际胜场 − 0.5」求解并标记 BoundAtLeast，全负按「实际胜场 + 0.5」求解并标记 BoundAtMost，
//     避免无解（真实水平趋于无穷）。
//   - 结果取整，且不小于 0（与排位分下限一致）。
func PerformanceRating(samples []PerformanceSample, minSamples int) (rating int, bound Bound, ok bool) {
	if len(samples) == 0 || len(samples) < minSamples {
		return 0, BoundExact, false
	}
	wins, maxOpponent := 0, samples[0].OpponentRating
	for _, sample := range samples {
		if sample.Won {
			wins++
		}
		maxOpponent = max(maxOpponent, sample.OpponentRating)
	}
	target := float64(wins)
	switch wins {
	case len(samples):
		target -= 0.5
		bound = BoundAtLeast
	case 0:
		target += 0.5
		bound = BoundAtMost
	}

	expected := func(r int) float64 {
		total := 0.0
		for _, sample := range samples {
			total += RaceWinProbability(RackWinProbability(r, sample.OpponentRating), sample.RaceLength)
		}
		return total
	}

	// 期望胜场随 R 单调递增：在整数分上二分，找到刚好达到目标的最小分，再取离目标更近的整数。
	lo, hi := 0, max(maxOpponent, 0)+performanceSearchStep
	if expected(lo) >= target {
		return 0, bound, true
	}
	for i := 0; i < performanceSearchMaxExpansions && expected(hi) < target; i++ {
		hi += performanceSearchStep
	}
	for hi-lo > 1 {
		mid := lo + (hi-lo)/2
		if expected(mid) < target {
			lo = mid
		} else {
			hi = mid
		}
	}
	if target-expected(lo) < expected(hi)-target {
		return lo, bound, true
	}
	return hi, bound, true
}
