package model

import "testing"

func TestBuildSnookerRoundStateStartsClearanceAfterFoulFollowingLastRed(t *testing.T) {
	actions := []MatchAction{
		{RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 1},
		{RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 1},
		{RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 1},
		{RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 1},
		{RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 1},
		{RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 1},
		{RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 1},
		{RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 1},
		{RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 1},
		{RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 1},
		{RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 1},
		{RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 1},
		{RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 1},
		{RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 1},
		{RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 1},
		{RoundNo: 1, ActionType: "foul", Actor: 2, ScoreChange: 4},
	}

	state := BuildSnookerRoundState(actions, 1)
	if !state.ClearanceStarted {
		t.Fatalf("expected clearance to start after foul following last red")
	}
	if state.ExpectedClearanceScore != 2 {
		t.Fatalf("expected next clearance ball to be yellow, got %d", state.ExpectedClearanceScore)
	}
}
