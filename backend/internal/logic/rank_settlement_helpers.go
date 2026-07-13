package logic

import (
	"time"

	"chasing_points/internal/model"
)

func normalizeAchievementType(achievementType string) string {
	switch achievementType {
	case "break_clear":
		return "break_and_run"
	case "continue_clear":
		return "run_out"
	case "small_gold":
		return "golden_break"
	case "big_gold":
		return "nine_on_break"
	default:
		return achievementType
	}
}

func calculateAchievementScoresByActor(rounds []model.MatchRound, rewardMap map[string]int) (int, int) {
	actor1 := 0
	actor2 := 0

	for _, round := range rounds {
		if round.Winner == nil {
			continue
		}
		normalized := normalizeAchievementType(round.WinType)
		if normalized == "" || normalized == "normal" || normalized == "start" {
			continue
		}
		reward := rewardMap[normalized]
		if reward <= 0 {
			continue
		}

		if *round.Winner == 1 {
			actor1 += reward
		} else if *round.Winner == 2 {
			actor2 += reward
		}
	}

	return actor1, actor2
}

func applySettlementToRanking(ranking *model.UserRanking, settlement RankSettlementResult) {
	ranking.RankScore = settlement.AfterScore
	ranking.RankLevel = settlement.AfterLevel
	ranking.TotalWins = settlement.TotalWins
	ranking.TotalLosses = settlement.TotalLosses
	ranking.CurrentStreak = settlement.CurrentStreak
	ranking.MaxStreak = settlement.MaxStreak
}

func resolveRankChangeEffectiveAt(match *model.Match) time.Time {
	if match == nil {
		return time.Now()
	}
	if match.EndTime != nil && !match.EndTime.IsZero() {
		return *match.EndTime
	}
	if !match.MatchTime.IsZero() {
		return match.MatchTime
	}
	return time.Now()
}

func buildRankChangeLog(matchId, userId int64, gameType int, result string, effectiveAt time.Time, settlement RankSettlementResult) model.RankChangeLog {
	return model.RankChangeLog{
		UserId:           userId,
		MatchId:          matchId,
		ChangeType:       "match_result",
		GameType:         gameType,
		Result:           result,
		BaseScore:        settlement.BaseScore,
		AchievementScore: settlement.AchievementScore,
		FinalChange:      settlement.FinalChange,
		BeforeScore:      settlement.BeforeScore,
		AfterScore:       settlement.AfterScore,
		BeforeLevel:      settlement.BeforeLevel,
		AfterLevel:       settlement.AfterLevel,
		Remark:           buildRankSettlementRemark(settlement),
		EffectiveAt:      effectiveAt,
	}
}

func resultLabel(isWin bool, isDraw bool) string {
	if isDraw {
		return "draw"
	}
	if isWin {
		return "win"
	}
	return "lose"
}

func rankSettlementDayRange(at time.Time) (time.Time, time.Time) {
	if at.IsZero() {
		at = time.Now()
	}
	location := at.Location()
	dayStart := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, location)
	return dayStart, dayStart.Add(24 * time.Hour)
}
