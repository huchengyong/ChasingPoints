package logic

import (
	"encoding/json"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

const (
	defaultDailyPositiveCap     = 500
	defaultMemberAchievementDailyCap = 200
	minimumWinnerBaseScore      = 2
	lossMinimumDeduction        = -2
	sameOpponentThirdMatchRate  = 80
	sameOpponentRepeatMatchRate = 30
)

const DefaultDailyPositiveCap = defaultDailyPositiveCap

type RankSettlementResult struct {
	BeforeScore            int
	AfterScore             int
	BeforeLevel            int
	AfterLevel             int
	BaseScore              int
	AchievementScore       int
	FinalChange            int
	TotalWins              int
	TotalLosses            int
	CurrentStreak          int
	MaxStreak              int
	LossFloorAdjustment    int
	SameOpponentAdjustment int
	DailyCapAdjustment     int
	MemberAchievementCapAdjustment int
	Details                []types.RankDetail
}

type RankSettlementPolicy struct {
	TodayPositiveGain        int
	DailyPositiveCap         int
	SameOpponentMatchesToday int
	TodayMemberAchievementGain int
	DailyMemberAchievementCap  int
	MemberLevel                int
	MemberActive               bool
	MemberMultiplierPercent     int
	OrdinaryUserAchievementEnabled bool
	CompletedRounds             int
	OpponentCurrentRankScore    int
}

type rankSettlementRemark struct {
	LossFloorAdjustment    int `json:"loss_floor_adjustment,omitempty"`
	SameOpponentAdjustment int `json:"same_opponent_adjustment,omitempty"`
	DailyCapAdjustment     int `json:"daily_cap_adjustment,omitempty"`
	MemberAchievementCapAdjustment int `json:"member_achievement_cap_adjustment,omitempty"`
}

type RankSettlementService struct {
	rankingModel *model.RankingModel
}

func NewRankSettlementService(rankingModel *model.RankingModel) *RankSettlementService {
	return &RankSettlementService{rankingModel: rankingModel}
}

func (s *RankSettlementService) Settle(ranking *model.UserRanking, isWin bool, achievementScore int) RankSettlementResult {
	return s.SettleWithPolicy(ranking, isWin, achievementScore, RankSettlementPolicy{
		CompletedRounds:          1,
		OpponentCurrentRankScore: 1 << 30,
	})
}

func (s *RankSettlementService) SettleWithPolicy(
	ranking *model.UserRanking,
	isWin bool,
	achievementScore int,
	policy RankSettlementPolicy,
) RankSettlementResult {
	current := model.UserRanking{}
	if ranking != nil {
		current = *ranking
	}

	beforeLevel := current.RankLevel
	if beforeLevel == 0 {
		beforeLevel = s.calculateLevel(current.RankScore)
	}

	countInStats := shouldCountRankStats(policy)
	completedRounds := normalizeCompletedRounds(policy.CompletedRounds)
	theoreticalWinBase := winnerBaseScoreByCompletedRounds(completedRounds)
	theoreticalLossDeduction := loserBaseDeductionFromWinnerBase(theoreticalWinBase)
	baseScore := 0
	if isWin {
		baseScore = theoreticalWinBase
		if theoreticalLossDeduction > 0 {
			if policy.OpponentCurrentRankScore < 0 {
				baseScore = theoreticalWinBase
			} else if policy.OpponentCurrentRankScore == 0 {
				baseScore = minimumWinnerBaseScore
			} else {
				opponentDeductible := minInt(policy.OpponentCurrentRankScore, theoreticalLossDeduction)
				baseScore = theoreticalWinBase * opponentDeductible / theoreticalLossDeduction
				if baseScore < minimumWinnerBaseScore {
					baseScore = minimumWinnerBaseScore
				}
			}
		}
		if countInStats {
			current.TotalWins++
			if current.CurrentStreak >= 0 {
				current.CurrentStreak++
			} else {
				current.CurrentStreak = 1
			}
			if current.CurrentStreak > current.MaxStreak {
				current.MaxStreak = current.CurrentStreak
			}
		}
	} else {
		if current.RankScore > 0 && theoreticalLossDeduction > 0 {
			baseScore = -minInt(current.RankScore, theoreticalLossDeduction)
		}
		if countInStats {
			current.TotalLosses++
			if current.CurrentStreak <= 0 {
				current.CurrentStreak--
			} else {
				current.CurrentStreak = -1
			}
		}
	}

	finalChange := baseScore + achievementScore
	lossFloorAdjustment := 0
	sameOpponentAdjustment := 0
	dailyCapAdjustment := 0
	memberAchievementCapAdjustment := 0

	if isWin {
		memberAchievementRemaining := remainingDailyPositiveGain(policy.TodayMemberAchievementGain, policy.DailyMemberAchievementCap)
		if achievementScore > memberAchievementRemaining {
			memberAchievementCapAdjustment = memberAchievementRemaining - achievementScore
			achievementScore = memberAchievementRemaining
			finalChange = baseScore + achievementScore
		}

		sameOpponentRate := sameOpponentRankGainRate(policy.SameOpponentMatchesToday)
		if finalChange > 0 && sameOpponentRate < 100 {
			scaled := scalePositiveGain(finalChange, sameOpponentRate)
			sameOpponentAdjustment = scaled - finalChange
			finalChange = scaled
		}

		remainingGain := remainingDailyPositiveGain(policy.TodayPositiveGain, policy.DailyPositiveCap)
		if finalChange > remainingGain {
			dailyCapAdjustment = remainingGain - finalChange
			finalChange = remainingGain
		}
	} else if !countInStats {
		sameOpponentAdjustment = -finalChange
		finalChange = 0
	} else if current.RankScore <= 0 && finalChange > 0 {
		lossFloorAdjustment = -finalChange
		finalChange = 0
	} else if current.RankScore > 0 {
		minimumFinalChange := baseScore
		if minimumFinalChange < lossMinimumDeduction {
			minimumFinalChange = lossMinimumDeduction
		}
		if finalChange > minimumFinalChange {
			lossFloorAdjustment = minimumFinalChange - finalChange
			finalChange = minimumFinalChange
		}
	}

	current.RankScore += finalChange
	if current.RankScore < 0 {
		current.RankScore = 0
	}
	current.RankLevel = s.calculateLevel(current.RankScore)

	details := []types.RankDetail{
		{Label: "基础分", Value: baseScore},
	}
	if achievementScore != 0 {
		label := "会员特殊战绩分"
		details = append(details, types.RankDetail{Label: label, Value: achievementScore})
	}
	if memberAchievementCapAdjustment != 0 {
		details = append(details, types.RankDetail{Label: "会员特殊战绩每日封顶", Value: memberAchievementCapAdjustment})
	}
	if lossFloorAdjustment != 0 {
		details = append(details, types.RankDetail{Label: "失败保底", Value: lossFloorAdjustment})
	}
	if sameOpponentAdjustment != 0 {
		label := "同对手衰减"
		if sameOpponentAdjustment > 0 {
			label = "同对手免扣"
		}
		details = append(details, types.RankDetail{Label: label, Value: sameOpponentAdjustment})
	}
	if dailyCapAdjustment != 0 {
		details = append(details, types.RankDetail{Label: "每日封顶", Value: dailyCapAdjustment})
	}

	return RankSettlementResult{
		BeforeScore:            rankingScoreOrZero(ranking),
		AfterScore:             current.RankScore,
		BeforeLevel:            beforeLevel,
		AfterLevel:             current.RankLevel,
		BaseScore:              baseScore,
		AchievementScore:       achievementScore,
		FinalChange:            finalChange,
		TotalWins:              current.TotalWins,
		TotalLosses:            current.TotalLosses,
		CurrentStreak:          current.CurrentStreak,
		MaxStreak:              current.MaxStreak,
		LossFloorAdjustment:    lossFloorAdjustment,
		SameOpponentAdjustment: sameOpponentAdjustment,
		DailyCapAdjustment:     dailyCapAdjustment,
		MemberAchievementCapAdjustment: memberAchievementCapAdjustment,
		Details:                details,
	}
}

func (s *RankSettlementService) calculateLevel(score int) int {
	if s != nil && s.rankingModel != nil {
		return s.rankingModel.CalculateLevel(score)
	}

	switch {
	case score >= 2000:
		return 5
	case score >= 1500:
		return 4
	case score >= 1000:
		return 3
	case score >= 500:
		return 2
	default:
		return 1
	}
}

func rankingScoreOrZero(ranking *model.UserRanking) int {
	if ranking == nil {
		return 0
	}
	return ranking.RankScore
}

func normalizeCompletedRounds(completedRounds int) int {
	if completedRounds <= 0 {
		return 1
	}
	return completedRounds
}

func winnerBaseScoreByCompletedRounds(completedRounds int) int {
	completedRounds = normalizeCompletedRounds(completedRounds)
	score := 8
	if completedRounds > 1 {
		score += (minInt(completedRounds, 5) - 1) * 3
	}
	if completedRounds > 5 {
		score += (minInt(completedRounds, 10) - 5) * 2
	}
	if completedRounds > 10 {
		score += minInt(completedRounds, 20) - 10
	}
	if score > 40 {
		return 40
	}
	return score
}

func loserBaseDeductionFromWinnerBase(winnerBase int) int {
	if winnerBase <= 0 {
		return 0
	}
	return winnerBase / 2
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func sameOpponentRankGainRate(priorCompletedMatches int) int {
	switch {
	case priorCompletedMatches < 0:
		return 100
	case priorCompletedMatches < 2:
		return 100
	case priorCompletedMatches == 2:
		return sameOpponentThirdMatchRate
	case priorCompletedMatches <= 5:
		return sameOpponentRepeatMatchRate
	default:
		return 0
	}
}

func scalePositiveGain(value int, rate int) int {
	if value <= 0 || rate >= 100 {
		return value
	}
	if rate <= 0 {
		return 0
	}
	return value * rate / 100
}

func remainingDailyPositiveGain(todayPositiveGain int, cap int) int {
	if cap <= 0 {
		cap = defaultDailyPositiveCap
	}
	remaining := cap - todayPositiveGain
	if remaining < 0 {
		return 0
	}
	return remaining
}

func shouldCountRankStats(policy RankSettlementPolicy) bool {
	return sameOpponentRankGainRate(policy.SameOpponentMatchesToday) > 0
}

func buildRankSettlementRemark(settlement RankSettlementResult) string {
	remark := rankSettlementRemark{
		LossFloorAdjustment:    settlement.LossFloorAdjustment,
		SameOpponentAdjustment: settlement.SameOpponentAdjustment,
		DailyCapAdjustment:     settlement.DailyCapAdjustment,
		MemberAchievementCapAdjustment: settlement.MemberAchievementCapAdjustment,
	}
	if remark == (rankSettlementRemark{}) {
		return "{}"
	}

	data, err := json.Marshal(remark)
	if err != nil {
		return ""
	}
	return string(data)
}

func BuildRankSettlementRemark(settlement RankSettlementResult) string {
	return buildRankSettlementRemark(settlement)
}

func parseRankSettlementRemark(raw string) rankSettlementRemark {
	if raw == "" {
		return rankSettlementRemark{}
	}

	var remark rankSettlementRemark
	if err := json.Unmarshal([]byte(raw), &remark); err != nil {
		return rankSettlementRemark{}
	}
	return remark
}
