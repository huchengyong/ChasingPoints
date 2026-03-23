package model

type SnookerRoundState struct {
	RedBallCount           int   `json:"red_ball_count"`
	ClearanceStarted       bool  `json:"clearance_started"`
	ClearedColors          []int `json:"cleared_colors"`
	ExpectedClearanceScore int   `json:"expected_clearance_score"`
	ClearanceCompleted     bool  `json:"clearance_completed"`
}

func BuildSnookerRoundState(actions []MatchAction, roundNo int) SnookerRoundState {
	state := SnookerRoundState{
		ClearedColors: make([]int, 0, 6),
	}
	if roundNo <= 0 {
		return state
	}

	phase := 0 // 0=红球阶段 1=等待最后一红后的收尾彩球 2=清彩阶段
	for _, action := range actions {
		if action.RoundNo != roundNo {
			continue
		}

		switch action.ActionType {
		case "score":
			switch action.ScoreChange {
			case 1:
				if state.RedBallCount < 15 {
					state.RedBallCount++
					if state.RedBallCount == 15 {
						phase = 1
					}
				}
				continue
			case 2, 3, 4, 5, 6, 7:
				if phase == 1 {
					phase = 2
					state.ClearanceStarted = true
					state.ExpectedClearanceScore = 2
					continue
				}
				if phase == 2 {
					expected := state.ExpectedClearanceScore
					if expected == 0 {
						expected = nextSnookerClearanceScore(len(state.ClearedColors))
					}
					if action.ScoreChange != expected {
						continue
					}
					state.ClearedColors = append(state.ClearedColors, action.ScoreChange)
					next := nextSnookerClearanceScore(len(state.ClearedColors))
					state.ExpectedClearanceScore = next
					if next == 0 {
						state.ClearanceCompleted = true
					}
				}
			}
		case "foul":
			if phase == 1 {
				phase = 2
				state.ClearanceStarted = true
				state.ExpectedClearanceScore = 2
			}
		}
	}

	if phase == 2 {
		state.ClearanceStarted = true
		if !state.ClearanceCompleted && state.ExpectedClearanceScore == 0 {
			state.ExpectedClearanceScore = nextSnookerClearanceScore(len(state.ClearedColors))
		}
	}

	return state
}

func nextSnookerClearanceScore(clearedCount int) int {
	order := []int{2, 3, 4, 5, 6, 7}
	if clearedCount < 0 || clearedCount >= len(order) {
		return 0
	}
	return order[clearedCount]
}

func containsSnookerColor(colors []int, score int) bool {
	for _, color := range colors {
		if color == score {
			return true
		}
	}
	return false
}
