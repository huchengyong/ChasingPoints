package logic

import (
	"fmt"

	"chasing_points/internal/model"
)

func calculateSnookerAchievementScoresByActor(actions []model.MatchAction, rewardMap map[string]int) (int, int) {
	player1, player2, err := calculateSnookerAchievementScoresByActorStrict(actions, rewardMap)
	if err != nil {
		return 0, 0
	}
	return player1, player2
}

func calculateSnookerAchievementScoresByActorStrict(actions []model.MatchAction, rewardMap map[string]int) (int, int, error) {
	breaks, version2, err := collectSnookerBreaksByActor(actions)
	if err != nil {
		return 0, 0, err
	}
	if version2 {
		totals := map[int]int{1: 0, 2: 0}
		for actor, scores := range breaks {
			for _, score := range scores {
				totals[actor] += scoreSnookerBreak(score, rewardMap)
			}
		}
		return totals[1], totals[2], nil
	}

	if len(actions) == 0 {
		return 0, 0, nil
	}
	totals := map[int]int{1: 0, 2: 0}
	currentRound := 0
	currentActor := 0
	currentBreak := 0
	finalizeCurrentBreak := func() {
		if currentActor == 1 || currentActor == 2 {
			totals[currentActor] += scoreSnookerBreak(currentBreak, rewardMap)
		}
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
	return totals[1], totals[2], nil
}

func collectSnookerBreaksByActor(actions []model.MatchAction) (map[int][]int, bool, error) {
	hasVersion2 := false
	for _, action := range actions {
		if action.ActionType == model.MatchActionTypeSnookerStroke || action.ActionType == model.MatchActionTypeSnookerFrameAction {
			hasVersion2 = true
			break
		}
	}
	breaks := map[int][]int{1: {}, 2: {}}
	if !hasVersion2 {
		return breaks, false, nil
	}
	type visitKey struct {
		round int
		actor int
		visit int
	}
	totals := make(map[visitKey]int)
	order := make([]visitKey, 0)
	seen := make(map[visitKey]bool)
	for _, action := range actions {
		if action.IsUndone != 0 {
			continue
		}
		switch action.ActionType {
		case model.MatchActionTypeSnookerStroke:
			event, err := model.DecodeSnookerEvent(action.ExtraData)
			if err != nil {
				return nil, true, fmt.Errorf("action %d: %w", action.Id, err)
			}
			if event.Kind != model.SnookerEventKindStroke || event.Actor != action.Actor {
				return nil, true, fmt.Errorf("action %d: snooker stroke mismatch", action.Id)
			}
			key := visitKey{round: action.RoundNo, actor: action.Actor, visit: event.VisitNo}
			if !seen[key] {
				seen[key] = true
				order = append(order, key)
			}
			switch event.Outcome {
			case model.SnookerOutcomePot:
				if action.ScoreChange <= 0 {
					return nil, true, fmt.Errorf("action %d: invalid pot score", action.Id)
				}
				totals[key] += action.ScoreChange
			case model.SnookerOutcomeNoScore:
				if action.ScoreChange != 0 {
					return nil, true, fmt.Errorf("action %d: no-score changed score", action.Id)
				}
			case model.SnookerOutcomeFoul:
				if action.ScoreChange < 4 || action.ScoreChange > 7 || action.ScoreChange != event.Penalty {
					return nil, true, fmt.Errorf("action %d: invalid foul score", action.Id)
				}
			}
		case model.MatchActionTypeSnookerFrameAction:
			event, err := model.DecodeSnookerEvent(action.ExtraData)
			if err != nil || event.Kind != model.SnookerEventKindFrameAction || event.Actor != action.Actor || action.ScoreChange != 0 {
				return nil, true, fmt.Errorf("action %d: invalid snooker frame action", action.Id)
			}
		case "round_start", "match_end", "undo":
			continue
		case "score", "foul", "win":
			return nil, true, fmt.Errorf("action %d: legacy scoring mixed with snooker version 2", action.Id)
		default:
			return nil, true, fmt.Errorf("action %d: unsupported action type %s", action.Id, action.ActionType)
		}
	}
	for _, key := range order {
		if totals[key] > 0 {
			breaks[key.actor] = append(breaks[key.actor], totals[key])
		}
	}
	return breaks, true, nil
}

func CalculateSnookerAchievementScoresByActor(actions []model.MatchAction, rewardMap map[string]int) (int, int) {
	return calculateSnookerAchievementScoresByActor(actions, rewardMap)
}

func CalculateSnookerAchievementScoresByActorStrict(actions []model.MatchAction, rewardMap map[string]int) (int, int, error) {
	return calculateSnookerAchievementScoresByActorStrict(actions, rewardMap)
}

type SnookerBreakAchievementCounts struct {
	FiftyPlus int
	Centuries int
	Break147  int
}

func CalculateSnookerBreakAchievementCounts(actions []model.MatchAction) (map[int]SnookerBreakAchievementCounts, error) {
	breaks, version2, err := collectSnookerBreaksByActor(actions)
	if err != nil {
		return nil, err
	}
	if !version2 {
		return map[int]SnookerBreakAchievementCounts{}, nil
	}
	result := map[int]SnookerBreakAchievementCounts{1: {}, 2: {}}
	for actor, scores := range breaks {
		counts := result[actor]
		for _, score := range scores {
			if score >= 50 {
				counts.FiftyPlus++
			}
			if score >= 100 {
				counts.Centuries++
			}
			if score == 147 {
				counts.Break147++
			}
		}
		result[actor] = counts
	}
	return result, nil
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
