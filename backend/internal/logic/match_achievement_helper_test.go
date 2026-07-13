package logic

import (
	"testing"

	"chasing_points/internal/model"
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

func TestBuildMatchAchievementPayload(t *testing.T) {
	payload := buildMatchAchievementPayload([]model.MatchAchievement{
		{AchievementType: "break_clear", Count: 1},
		{AchievementType: "continue_clear", Count: 2},
		{AchievementType: "golden_break", Count: 3},
		{AchievementType: "nine_on_break", Count: 4},
		{AchievementType: "break_50", Count: 5},
		{AchievementType: "break_100", Count: 6},
		{AchievementType: "break_147", Count: 7},
	})

	if payload.BreakClear != 1 {
		t.Fatalf("expected break_clear=1, got %d", payload.BreakClear)
	}
	if payload.ContinueClear != 2 {
		t.Fatalf("expected continue_clear=2, got %d", payload.ContinueClear)
	}
	if payload.GoldenBreak != 3 {
		t.Fatalf("expected golden_break=3, got %d", payload.GoldenBreak)
	}
	if payload.NineOnBreak != 4 {
		t.Fatalf("expected nine_on_break=4, got %d", payload.NineOnBreak)
	}
	if payload.Break50 != 5 {
		t.Fatalf("expected break_50=5, got %d", payload.Break50)
	}
	if payload.Break100 != 6 {
		t.Fatalf("expected break_100=6, got %d", payload.Break100)
	}
	if payload.Break147 != 7 {
		t.Fatalf("expected break_147=7, got %d", payload.Break147)
	}
}
