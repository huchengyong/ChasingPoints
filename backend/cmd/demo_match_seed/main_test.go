package main

import "testing"

func TestBuildDemoScenariosCoversCoreMatchOutcomes(t *testing.T) {
	scenarios := buildDemoScenarios()
	if len(scenarios) < 20 {
		t.Fatalf("expected broad demo coverage, got %d scenarios", len(scenarios))
	}

	user1WinsOverUser2 := 0
	hasUser2Win := false
	gameTypes := map[int]bool{}
	winTypes := map[string]bool{}

	for _, scenario := range scenarios {
		gameTypes[scenario.GameType] = true
		for _, round := range scenario.Rounds {
			winTypes[round.WinType] = true
		}
		if scenario.Player1ID == 1 && scenario.Player2ID == 2 && scenario.WinnerID() == 1 {
			user1WinsOverUser2++
		}
		if scenario.Player1ID == 1 && scenario.Player2ID == 2 && scenario.WinnerID() == 2 {
			hasUser2Win = true
		}
	}

	if user1WinsOverUser2 < 10 {
		t.Fatalf("expected user 1 to beat user 2 at least 10 times, got %d", user1WinsOverUser2)
	}
	if !hasUser2Win {
		t.Fatal("expected at least one user 2 win over user 1")
	}
	for _, gameType := range []int{1, 2, 3, 4} {
		if !gameTypes[gameType] {
			t.Fatalf("expected game type %d coverage", gameType)
		}
	}
	for _, winType := range []string{"normal", "break_clear", "continue_clear", "golden_break", "nine_on_break", "break_50", "break_100", "break_147"} {
		if !winTypes[winType] {
			t.Fatalf("expected win type %q coverage", winType)
		}
	}
}
