package logic

import (
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func TestBuildMatchSummaryForAmericanNine(t *testing.T) {
	winner1 := 1
	winner2 := 2
	rounds := []model.MatchRound{
		{Winner: &winner1, WinType: "normal"},
		{Winner: &winner1, WinType: "small_gold"},
		{Winner: &winner2, WinType: "normal"},
		{Winner: &winner1, WinType: "big_gold"},
	}

	highlights, stats := buildMatchSummary(
		4,
		1,
		3,
		1,
		0.6,
		0.4,
		0,
		0,
		0,
		"2026-03-14 20:00",
		rounds,
		nil,
		types.MatchAchievement{},
	)

	if !hasSummaryItem(highlights, "小金次数", "1 次") {
		t.Fatalf("expected american nine highlights to include small gold count, got %#v", highlights)
	}
	if !hasSummaryItem(highlights, "大金次数", "1 次") {
		t.Fatalf("expected american nine highlights to include big gold count, got %#v", highlights)
	}
	if !hasSummaryItem(stats, "总局数", "4 局") {
		t.Fatalf("expected american nine stats to include total rounds, got %#v", stats)
	}
	if hasSummaryLabel(stats, "总得分") || hasSummaryLabel(highlights, "最高连续得分") {
		t.Fatalf("american nine summary should not reuse point-based nine-ball metrics, highlights=%#v stats=%#v", highlights, stats)
	}
}

func TestBuildMatchSummaryForSnookerUsesCurrentMatchBreaks(t *testing.T) {
	actions := []model.MatchAction{
		{RoundNo: 1, Actor: 2, ActionType: "score", ScoreChange: 1},
		{RoundNo: 1, Actor: 2, ActionType: "score", ScoreChange: 7},
		{RoundNo: 1, Actor: 2, ActionType: "score", ScoreChange: 1},
		{RoundNo: 1, Actor: 2, ActionType: "score", ScoreChange: 7},
		{RoundNo: 1, Actor: 2, ActionType: "score", ScoreChange: 1},
		{RoundNo: 1, Actor: 2, ActionType: "score", ScoreChange: 7},
		{RoundNo: 1, Actor: 2, ActionType: "score", ScoreChange: 7},
		{RoundNo: 1, Actor: 2, ActionType: "score", ScoreChange: 1},
		{RoundNo: 1, Actor: 2, ActionType: "score", ScoreChange: 1},
		{RoundNo: 1, Actor: 2, ActionType: "score", ScoreChange: 7},
		{RoundNo: 1, Actor: 2, ActionType: "score", ScoreChange: 1},
		{RoundNo: 1, Actor: 2, ActionType: "score", ScoreChange: 7},
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 1, Actor: 1, ActionType: "foul", ScoreChange: 7},
	}

	highlights, stats := buildMatchSummary(
		1,
		1,
		16,
		55,
		0.5,
		0.5,
		81,
		129,
		0,
		"2026-03-15 19:31:52",
		nil,
		actions,
		types.MatchAchievement{},
	)

	if !hasSummaryItem(highlights, "最高单杆", "16 分") {
		t.Fatalf("expected snooker highlights to use current match break, got %#v", highlights)
	}
	if !hasSummaryItem(stats, "我的最高单杆", "16 分") {
		t.Fatalf("expected snooker stats to use current match break, got %#v", stats)
	}
	if !hasSummaryItem(stats, "对手最高单杆", "48 分") {
		t.Fatalf("expected opponent snooker stats to use current match break, got %#v", stats)
	}
	if !hasSummaryItem(stats, "我的50+次数", "0 次") {
		t.Fatalf("expected snooker stats to include my fifty-plus count, got %#v", stats)
	}
	if !hasSummaryItem(stats, "对手50+次数", "0 次") {
		t.Fatalf("expected snooker stats to include opponent fifty-plus count, got %#v", stats)
	}
	if !hasSummaryItem(stats, "我的破百次数", "0 次") {
		t.Fatalf("expected snooker stats to include my century count, got %#v", stats)
	}
	if !hasSummaryItem(stats, "对手破百次数", "0 次") {
		t.Fatalf("expected snooker stats to include opponent century count, got %#v", stats)
	}
	if hasSummaryLabel(highlights, "我的红球进球") || hasSummaryLabel(stats, "我的红球进球") {
		t.Fatalf("snooker summary should no longer expose red-ball pot counts, highlights=%#v stats=%#v", highlights, stats)
	}
	if hasSummaryItem(highlights, "最高单杆", "81 分") || hasSummaryItem(stats, "对手最高单杆", "129 分") {
		t.Fatalf("snooker summary should not reuse historical max score, highlights=%#v stats=%#v", highlights, stats)
	}
}

func TestBuildMatchSummaryForSnookerIgnoresUnfinishedFrameActions(t *testing.T) {
	winner1 := 1
	rounds := []model.MatchRound{
		{RoundNo: 1, Winner: &winner1, WinType: "normal"},
	}
	actions := []model.MatchAction{
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 100},
	}

	highlights, stats := buildMatchSummary(
		1,
		1,
		1,
		0,
		0.5,
		0.5,
		0,
		0,
		0,
		"2026-03-17 20:00",
		rounds,
		actions,
		types.MatchAchievement{},
	)

	if !hasSummaryItem(highlights, "最高单杆", "16 分") {
		t.Fatalf("expected settled-frame break in highlights, got %#v", highlights)
	}
	if hasSummaryItem(highlights, "最高单杆", "100 分") {
		t.Fatalf("unfinished frame break should not leak into highlights, got %#v", highlights)
	}
	if !hasSummaryItem(stats, "我的最高单杆", "16 分") {
		t.Fatalf("expected settled-frame break in stats, got %#v", stats)
	}
}

func TestBuildMatchSummaryForSnookerTracksFiftyPlusAndCenturies(t *testing.T) {
	actions := []model.MatchAction{
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 1, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 1},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 2},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 3},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 4},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 5},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 6},
		{RoundNo: 2, Actor: 1, ActionType: "score", ScoreChange: 7},
		{RoundNo: 3, Actor: 2, ActionType: "score", ScoreChange: 1},
		{RoundNo: 3, Actor: 2, ActionType: "score", ScoreChange: 7},
		{RoundNo: 3, Actor: 2, ActionType: "score", ScoreChange: 1},
		{RoundNo: 3, Actor: 2, ActionType: "score", ScoreChange: 7},
		{RoundNo: 3, Actor: 2, ActionType: "score", ScoreChange: 1},
		{RoundNo: 3, Actor: 2, ActionType: "score", ScoreChange: 7},
		{RoundNo: 3, Actor: 2, ActionType: "score", ScoreChange: 1},
		{RoundNo: 3, Actor: 2, ActionType: "score", ScoreChange: 7},
		{RoundNo: 3, Actor: 2, ActionType: "foul", ScoreChange: 4},
	}

	highlights, stats := buildMatchSummary(
		1,
		1,
		2,
		1,
		0.5,
		0.5,
		0,
		0,
		0,
		"2026-03-18 12:15:00",
		nil,
		actions,
		types.MatchAchievement{},
	)

	if !hasSummaryItem(highlights, "最高单杆", "147 分") {
		t.Fatalf("expected snooker highlights to show current actor highest break, got %#v", highlights)
	}
	if !hasSummaryItem(highlights, "50+次数", "1 次") {
		t.Fatalf("expected snooker highlights to show current actor fifty-plus count, got %#v", highlights)
	}
	if !hasSummaryItem(highlights, "破百次数", "1 次") {
		t.Fatalf("expected snooker highlights to show current actor century count, got %#v", highlights)
	}
	if !hasSummaryItem(stats, "我的50+次数", "1 次") {
		t.Fatalf("expected snooker stats to show my fifty-plus count, got %#v", stats)
	}
	if !hasSummaryItem(stats, "我的破百次数", "1 次") {
		t.Fatalf("expected snooker stats to show my century count, got %#v", stats)
	}
	if !hasSummaryItem(stats, "对手50+次数", "0 次") {
		t.Fatalf("expected snooker stats to show opponent fifty-plus count, got %#v", stats)
	}
	if !hasSummaryItem(stats, "对手破百次数", "0 次") {
		t.Fatalf("expected snooker stats to show opponent century count, got %#v", stats)
	}
}

func TestBuildMatchSummaryForNineBallDoesNotCountOwnFoulsAsRun(t *testing.T) {
	actions := []model.MatchAction{
		{RoundNo: 1, Actor: 1, ActionType: "win", ScoreChange: 4},
		{RoundNo: 2, Actor: 1, ActionType: "foul", ScoreChange: 1},
		{RoundNo: 2, Actor: 1, ActionType: "foul", ScoreChange: 1},
	}

	highlights, _ := buildMatchSummary(
		2,
		1,
		4,
		2,
		0.5,
		0.5,
		0,
		0,
		0,
		"2026-03-15 18:51:34",
		nil,
		actions,
		types.MatchAchievement{},
	)

	if !hasSummaryItem(highlights, "最高连续得分", "4 分") {
		t.Fatalf("expected nine-ball highlight to ignore own fouls, got %#v", highlights)
	}
	if hasSummaryItem(highlights, "最高连续得分", "6 分") {
		t.Fatalf("nine-ball highlight should not add foul points to fouling actor, got %#v", highlights)
	}
}

func TestBuildMatchSummaryForEightBallUsesCurrentActorSpecialWins(t *testing.T) {
	winner1 := 1
	winner2 := 2
	rounds := []model.MatchRound{
		{Winner: &winner1, WinType: "break_clear"},
		{Winner: &winner2, WinType: "break_clear"},
		{Winner: &winner2, WinType: "break_clear"},
		{Winner: &winner1, WinType: "continue_clear"},
	}

	highlights, _ := buildMatchSummary(
		3,
		1,
		2,
		2,
		0.5,
		0.5,
		0,
		0,
		0,
		"2026-03-15 18:24:40",
		rounds,
		nil,
		types.MatchAchievement{
			BreakClear:    3,
			ContinueClear: 1,
		},
	)

	if !hasSummaryItem(highlights, "炸清次数", "1 次") {
		t.Fatalf("expected eight-ball highlight to use current actor break clear count, got %#v", highlights)
	}
	if !hasSummaryItem(highlights, "接清次数", "1 次") {
		t.Fatalf("expected eight-ball highlight to use current actor continue clear count, got %#v", highlights)
	}
	if hasSummaryItem(highlights, "炸清次数", "3 次") {
		t.Fatalf("eight-ball highlight should not use match-level break clear aggregate, got %#v", highlights)
	}
}

func hasSummaryItem(items []types.MatchSummaryItem, label string, value string) bool {
	for _, item := range items {
		if item.Label == label && item.Value == value {
			return true
		}
	}
	return false
}

func hasSummaryLabel(items []types.MatchSummaryItem, label string) bool {
	for _, item := range items {
		if item.Label == label {
			return true
		}
	}
	return false
}
