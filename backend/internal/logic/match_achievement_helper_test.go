package logic

import (
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func TestNormalizeStoredAchievementType(t *testing.T) {
	cases := map[string]string{
		"":             "",
		"normal":       "",
		"small_gold":   "golden_break",
		"big_gold":     "nine_on_break",
		"break_clear":  "break_and_run",
		"golden_break": "golden_break",
	}

	for input, expected := range cases {
		if got := normalizeStoredAchievementType(input); got != expected {
			t.Fatalf("normalizeStoredAchievementType(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestBuildMatchAchievementsByActor(t *testing.T) {
	player1, player2 := BuildMatchAchievementsByActor([]model.MatchAchievement{
		{Actor: 1, AchievementType: "break_clear", Count: 1},
		{Actor: 1, AchievementType: "break_and_run", Count: 1}, // 历史别名与 break_clear 累加
		{Actor: 1, AchievementType: "continue_clear", Count: 2},
		{Actor: 1, AchievementType: "golden_break", Count: 3},
		{Actor: 1, AchievementType: "nine_on_break", Count: 4},
		{Actor: 1, AchievementType: "break_50", Count: 5},
		{Actor: 1, AchievementType: "break_100", Count: 6},
		{Actor: 1, AchievementType: "break_147", Count: 7},
		{Actor: 2, AchievementType: "break_clear", Count: 1},
		{Actor: 2, AchievementType: "continue_clear", Count: 1},
	})

	if player1 != (types.MatchAchievement{BreakClear: 2, ContinueClear: 2, GoldenBreak: 3, NineOnBreak: 4, Break50: 5, Break100: 6, Break147: 7}) {
		t.Fatalf("player1 achievements = %+v", player1)
	}
	if player2.BreakClear != 1 || player2.ContinueClear != 1 {
		t.Fatalf("player2 achievements = %+v", player2)
	}
	if player2.GoldenBreak != 0 || player2.NineOnBreak != 0 || player2.Break50 != 0 {
		t.Fatalf("player2 must not inherit player1 achievements: %+v", player2)
	}
}
