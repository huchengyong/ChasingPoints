package logic

import "billiard_master/internal/model"

func calculateSnookerAchievementScoresByActor(actions []model.MatchAction, rewardMap map[string]int) (int, int) {
	if len(actions) == 0 {
		return 0, 0
	}

	totals := map[int]int{1: 0, 2: 0}
	currentRound := 0
	currentActor := 0
	currentBreak := 0

	finalizeCurrentBreak := func() {
		if currentActor != 1 && currentActor != 2 {
			currentActor = 0
			currentBreak = 0
			return
		}
		totals[currentActor] += scoreSnookerBreak(currentBreak, rewardMap)
		currentActor = 0
		currentBreak = 0
	}

	for _, action := range actions {
		if action.RoundNo != currentRound {
			finalizeCurrentBreak()
			currentRound = action.RoundNo
		}

		switch action.ActionType {
		case "score":
			if action.Actor != 1 && action.Actor != 2 {
				continue
			}
			if action.ScoreChange <= 0 {
				if currentActor == action.Actor {
					finalizeCurrentBreak()
				}
				continue
			}
			if currentActor != action.Actor {
				finalizeCurrentBreak()
				currentActor = action.Actor
			}
			currentBreak += action.ScoreChange
		case "foul":
			if action.Actor == currentActor {
				finalizeCurrentBreak()
			}
		case "win", "round_start":
			finalizeCurrentBreak()
		default:
			if action.Actor == currentActor {
				finalizeCurrentBreak()
			}
		}
	}

	finalizeCurrentBreak()

	return totals[1], totals[2]
}

func scoreSnookerBreak(breakScore int, rewardMap map[string]int) int {
	switch {
	case breakScore == 147:
		return rewardMap["break_147"]
	case breakScore >= 100:
		return rewardMap["break_100"]
	case breakScore >= 50:
		return rewardMap["break_50"]
	default:
		return 0
	}
}
