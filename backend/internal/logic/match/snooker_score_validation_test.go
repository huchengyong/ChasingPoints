package match

import (
	"testing"

	"chasing_points/internal/model"
)

func TestValidateSnookerScoreRejectsWrongClearanceOrder(t *testing.T) {
	state := model.SnookerRoundState{
		RedBallCount:           15,
		ClearanceStarted:       true,
		ExpectedClearanceScore: 2,
		ClearedColors:          []int{},
	}

	if err := validateSnookerScore(state, 3); err == nil {
		t.Fatalf("expected wrong clearance color to be rejected")
	}
	if err := validateSnookerScore(state, 2); err != nil {
		t.Fatalf("expected yellow to be accepted, got %v", err)
	}
}
