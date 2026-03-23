package logic

import "billiard_master/internal/model"

func filterSnookerActionsToCompletedRounds(actions []model.MatchAction, rounds []model.MatchRound) []model.MatchAction {
	if len(actions) == 0 || len(rounds) == 0 {
		return []model.MatchAction{}
	}

	completedRounds := make(map[int]struct{}, len(rounds))
	for _, round := range rounds {
		if round.Winner == nil {
			continue
		}
		completedRounds[round.RoundNo] = struct{}{}
	}
	if len(completedRounds) == 0 {
		return []model.MatchAction{}
	}

	filtered := make([]model.MatchAction, 0, len(actions))
	for _, action := range actions {
		if _, ok := completedRounds[action.RoundNo]; ok {
			filtered = append(filtered, action)
		}
	}

	return filtered
}

func filterSnookerActionsToRoundLimit(actions []model.MatchAction, maxRoundNo int) []model.MatchAction {
	if len(actions) == 0 || maxRoundNo <= 0 {
		return []model.MatchAction{}
	}

	filtered := make([]model.MatchAction, 0, len(actions))
	for _, action := range actions {
		if action.RoundNo <= maxRoundNo {
			filtered = append(filtered, action)
		}
	}
	return filtered
}
