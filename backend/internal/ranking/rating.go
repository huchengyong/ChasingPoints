// Package ranking 实现排位规则 V2 的纯计分：线上结算、初始分重放与审计共用，不访问数据库。
package ranking

import (
	"encoding/json"
	"math"
)

const (
	// InitialRating 是没有历史种子的用户在 V2 下的起始分（黄金段中部）。
	InitialRating = 1250
	// RatingScale：双方分差每达到该值，单局胜负比为 10:1。
	RatingScale = 2500.0
	// BaseK 是抢 1 的 K 值，抢 N 为 BaseK·√N。
	BaseK = 40.0
	// MaxRaceLength 限制参与计算的赛制长度，避免长自由局放大单场权重。
	MaxRaceLength = 9
	// PlacementMatches 是定级赛场数。
	PlacementMatches = 10
	// DefaultDailyPositiveCap 是未配置时的每日上涨上限。
	DefaultDailyPositiveCap = 500
	// StrengthBucketGap 是强弱对手相对分档的分差阈值。
	StrengthBucketGap = 250

	RulesVersionLegacy = 1
	RulesVersionV2     = 2

	StrengthBucketStronger = "rating_stronger"
	StrengthBucketEven     = "rating_even"
	StrengthBucketWeaker   = "rating_weaker"
)

// RaceLength 返回参与计算的赛制长度 N：抢 N 赛制取目标局数，自由局与旧赛制取双方赢局数的较大值，限制在 1–9。
func RaceLength(targetWins, player1Racks, player2Racks int) int {
	n := targetWins
	if n <= 0 {
		n = max(player1Racks, player2Racks)
	}
	return min(max(n, 1), MaxRaceLength)
}

// RackWinProbability 返回本人赢下单局的概率。
func RackWinProbability(rating, opponentRating int) float64 {
	return 1 / (1 + math.Pow(10, float64(opponentRating-rating)/RatingScale))
}

// RaceWinProbability 返回单局胜率为 p 时先赢 n 局的概率。
func RaceWinProbability(p float64, n int) float64 {
	n = max(n, 1)
	total := 0.0
	coefficient := 1.0 // C(n-1+k, k)
	for k := 0; k < n; k++ {
		if k > 0 {
			coefficient = coefficient * float64(n-1+k) / float64(k)
		}
		total += coefficient * math.Pow(p, float64(n)) * math.Pow(1-p, float64(k))
	}
	return total
}

// KFactor 返回赛制长度 n 的 K 值。
func KFactor(n int) float64 {
	return BaseK * math.Sqrt(float64(max(n, 1)))
}

// SameOpponentRate 返回当日同一对手此前已完成 prior 场时本场的计分比例（百分比）。
func SameOpponentRate(prior int) int {
	switch {
	case prior < 2:
		return 100
	case prior == 2:
		return 80
	case prior <= 5:
		return 30
	default:
		return 0
	}
}

// StrengthBucket 按结算前双方分差给出相对强弱分档。
func StrengthBucket(rating, opponentRating int) string {
	switch gap := opponentRating - rating; {
	case gap > StrengthBucketGap:
		return StrengthBucketStronger
	case gap < -StrengthBucketGap:
		return StrengthBucketWeaker
	default:
		return StrengthBucketEven
	}
}

// PlayerState 是一方在本场结算前的状态。
type PlayerState struct {
	Rating            int
	RatedMatches      int
	TodayPositiveGain int
}

func (s PlayerState) inPlacement() bool {
	return s.RatedMatches < PlacementMatches
}

// Input 描述一场有效排位的结算输入，Player1Won 为 false 表示 Player2 获胜。
type Input struct {
	Player1               PlayerState
	Player2               PlayerState
	Player1Won            bool
	RaceLength            int
	PriorPairMatchesToday int
	DailyPositiveCap      int
}

// Outcome 是一方的结算结果；各调整项之和等于 Change。
type Outcome struct {
	Rated            bool
	RaceLength       int
	Expected         float64
	Standard         int
	Placement        int
	OpponentDamping  int
	SameOpponent     int
	SameOpponentRate int
	DailyCap         int
	Floor            int
	Change           int
	Before           int
	After            int
}

// Settle 结算一场有效排位，返回双方结果。期望只从 Player1 视角计算一次，保证未调整时双方变化严格相反。
func Settle(in Input) (Outcome, Outcome) {
	in.Player1.Rating = max(in.Player1.Rating, 0)
	in.Player2.Rating = max(in.Player2.Rating, 0)
	n := max(in.RaceLength, 1)
	expected1 := RaceWinProbability(RackWinProbability(in.Player1.Rating, in.Player2.Rating), n)
	rate := SameOpponentRate(in.PriorPairMatchesToday)
	dailyCap := in.DailyPositiveCap
	if dailyCap <= 0 {
		dailyCap = DefaultDailyPositiveCap
	}
	p1 := settleSide(in.Player1, in.Player2, in.Player1Won, expected1, n, rate, dailyCap)
	p2 := settleSide(in.Player2, in.Player1, !in.Player1Won, 1-expected1, n, rate, dailyCap)
	return p1, p2
}

func settleSide(self, opponent PlayerState, won bool, expected float64, n, rate, dailyCap int) Outcome {
	out := Outcome{
		RaceLength:       n,
		Expected:         expected,
		SameOpponentRate: rate,
		Before:           self.Rating,
		After:            self.Rating,
	}
	if rate <= 0 {
		return out
	}
	out.Rated = true

	score := 0.0
	if won {
		score = 1
	}
	out.Standard = atLeastOne(int(math.Round(KFactor(n)*(score-expected))), won)
	change := out.Standard

	switch {
	case self.inPlacement():
		multiplied := atLeastOne(int(math.Round(float64(change)*2)), won)
		out.Placement = multiplied - change
		change = multiplied
	case opponent.inPlacement():
		damped := atLeastOne(int(math.Round(float64(change)*0.5)), won)
		out.OpponentDamping = damped - change
		change = damped
	}

	if rate < 100 {
		decayed := atLeastOne(int(math.Round(float64(change)*float64(rate)/100)), won)
		out.SameOpponent = decayed - change
		change = decayed
	}

	if change > 0 {
		remaining := max(dailyCap-self.TodayPositiveGain, 0)
		if change > remaining {
			out.DailyCap = remaining - change
			change = remaining
		}
	}

	if self.Rating+change < 0 {
		floored := -self.Rating
		out.Floor = floored - change
		change = floored
	}

	out.Change = change
	out.After = self.Rating + change
	return out
}

// atLeastOne 让有效排位的胜方至少 +1、负方至少 −1。
func atLeastOne(value int, won bool) int {
	if won && value < 1 {
		return 1
	}
	if !won && value > -1 {
		return -1
	}
	return value
}

// Remark 是 V2 结算写入 rank_change_logs.remark 的明细。
type Remark struct {
	Version          int     `json:"v"`
	Rated            bool    `json:"rated"`
	RaceLength       int     `json:"n,omitempty"`
	Expected         float64 `json:"e,omitempty"`
	Standard         int     `json:"std,omitempty"`
	Placement        int     `json:"plc,omitempty"`
	OpponentDamping  int     `json:"opc,omitempty"`
	SameOpponent     int     `json:"same,omitempty"`
	SameOpponentRate int     `json:"rate,omitempty"`
	DailyCap         int     `json:"cap,omitempty"`
	Floor            int     `json:"floor,omitempty"`
	SeedMatches      int     `json:"seed,omitempty"`
}

// RemarkFor 序列化一方的结算明细。
func RemarkFor(out Outcome) string {
	remark := Remark{
		Version:          RulesVersionV2,
		Rated:            out.Rated,
		RaceLength:       out.RaceLength,
		Expected:         math.Round(out.Expected*1000) / 1000,
		Standard:         out.Standard,
		Placement:        out.Placement,
		OpponentDamping:  out.OpponentDamping,
		SameOpponent:     out.SameOpponent,
		SameOpponentRate: out.SameOpponentRate,
		DailyCap:         out.DailyCap,
		Floor:            out.Floor,
	}
	data, _ := json.Marshal(remark)
	return string(data)
}

// ResetRemark 序列化规则切换记录，保存切换时的种子场次供审计恢复。
func ResetRemark(seedMatches int) string {
	data, _ := json.Marshal(Remark{Version: RulesVersionV2, SeedMatches: seedMatches})
	return string(data)
}

// ParseRemark 解析 V2 明细；不是 V2 明细时返回 ok=false。
func ParseRemark(raw string) (Remark, bool) {
	var remark Remark
	if raw == "" || json.Unmarshal([]byte(raw), &remark) != nil || remark.Version != RulesVersionV2 {
		return Remark{}, false
	}
	return remark, true
}
