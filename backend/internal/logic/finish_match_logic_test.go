package logic

import (
	"strings"
	"testing"
	"time"

	"billiard_master/internal/model"
)

func TestNormalizeAchievementType(t *testing.T) {
	cases := map[string]string{
		"break_clear":    "break_and_run",
		"continue_clear": "run_out",
		"small_gold":     "golden_break",
		"big_gold":       "nine_on_break",
		"golden_break":   "golden_break",
		"break_100":      "break_100",
	}

	for input, expected := range cases {
		if got := normalizeAchievementType(input); got != expected {
			t.Fatalf("normalizeAchievementType(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestCalculateAchievementScoresByActor(t *testing.T) {
	winner1 := 1
	winner2 := 2
	rounds := []model.MatchRound{
		{Winner: &winner1, WinType: "break_clear"},
		{Winner: &winner2, WinType: "continue_clear"},
		{Winner: &winner2, WinType: "golden_break"},
		{Winner: &winner1, WinType: "normal"},
	}

	rewardMap := map[string]int{
		"break_and_run": 15,
		"run_out":       10,
		"golden_break":  10,
	}

	actor1, actor2 := calculateAchievementScoresByActor(rounds, rewardMap)
	if actor1 != 15 {
		t.Fatalf("expected actor1 reward 15, got %d", actor1)
	}
	if actor2 != 20 {
		t.Fatalf("expected actor2 reward 20, got %d", actor2)
	}
}

func TestResolveReplayAchievementScoresPrefersCompletedRounds(t *testing.T) {
	winner1 := 1
	rounds := []model.MatchRound{
		{Winner: &winner1, WinType: "break_clear"},
	}
	storedAchievements := []model.MatchAchievement{
		{AchievementType: "break_clear", Count: 2},
	}
	rewardMap := map[string]int{
		"break_and_run": 15,
	}

	actor1, actor2 := resolveReplayAchievementScores(3, rounds, nil, storedAchievements, rewardMap)
	if actor1 != 15 || actor2 != 0 {
		t.Fatalf("expected replay scores from rounds only, got actor1=%d actor2=%d", actor1, actor2)
	}
}

func TestResolveReplayAchievementScoresFallsBackToStoredAchievements(t *testing.T) {
	storedAchievements := []model.MatchAchievement{
		{AchievementType: "break_clear", Count: 2},
		{AchievementType: "golden_break", Count: 1},
		{AchievementType: "normal", Count: 3},
	}
	rewardMap := map[string]int{
		"break_and_run": 15,
		"golden_break":  10,
	}

	actor1, actor2 := resolveReplayAchievementScores(3, nil, nil, storedAchievements, rewardMap)
	if actor1 != 40 || actor2 != 0 {
		t.Fatalf("expected fallback replay scores actor1=40 actor2=0, got actor1=%d actor2=%d", actor1, actor2)
	}
}

func TestCalculateSnookerAchievementScoresByActor(t *testing.T) {
	rewardMap := map[string]int{
		"break_50":  5,
		"break_100": 10,
		"break_147": 50,
	}

	actions := buildSnookerActionsForTest(
		1,
		1,
		[]int{1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 2, 3, 4, 5, 6, 7},
	)

	actor1, actor2 := calculateSnookerAchievementScoresByActor(actions, rewardMap)
	if actor1 != 50 || actor2 != 0 {
		t.Fatalf("expected actor1 reward 50 and actor2 reward 0, got actor1=%d actor2=%d", actor1, actor2)
	}
}

func TestResolveReplayAchievementScoresUsesSnookerActions(t *testing.T) {
	rewardMap := map[string]int{
		"break_50":  5,
		"break_100": 10,
		"break_147": 50,
	}

	winner1 := 1
	winner2 := 2
	actions := []model.MatchAction{}
	actions = append(actions, buildSnookerActionsForTest(1, 1, []int{30, 20})...)
	actions = append(actions, buildSnookerActionsForTest(2, 2, []int{60})...)

	rounds := []model.MatchRound{
		{RoundNo: 1, Winner: &winner1, WinType: "normal"},
		{RoundNo: 2, Winner: &winner2, WinType: "normal"},
	}

	actor1, actor2 := resolveReplayAchievementScores(1, rounds, actions, nil, rewardMap)
	if actor1 != 5 || actor2 != 5 {
		t.Fatalf("expected snooker replay rewards actor1=5 actor2=5, got actor1=%d actor2=%d", actor1, actor2)
	}
}

func TestResolveReplayAchievementScoresIgnoresUnfinishedSnookerFrame(t *testing.T) {
	rewardMap := map[string]int{
		"break_50":  5,
		"break_100": 10,
		"break_147": 50,
	}

	winner1 := 1
	rounds := []model.MatchRound{
		{RoundNo: 1, Winner: &winner1, WinType: "normal"},
	}
	actions := []model.MatchAction{}
	actions = append(actions, buildSnookerActionsForTest(1, 1, []int{30, 25})...)
	actions = append(actions, buildSnookerActionsForTest(2, 1, []int{100})...)

	actor1, actor2 := resolveReplayAchievementScores(1, rounds, actions, nil, rewardMap)
	if actor1 != 5 || actor2 != 0 {
		t.Fatalf("expected only settled frame reward actor1=5 actor2=0, got actor1=%d actor2=%d", actor1, actor2)
	}
}

func TestCalculateSnookerAchievementScoresByActorCountsMultipleBreaksInSameRound(t *testing.T) {
	rewardMap := map[string]int{
		"break_50":  5,
		"break_100": 10,
		"break_147": 50,
	}

	actions := []model.MatchAction{}
	actions = append(actions, buildSnookerActionsForTest(1, 1, []int{30, 25})...)
	actions = append(actions, model.MatchAction{RoundNo: 1, ActionType: "foul", Actor: 1, ScoreChange: 4})
	actions = append(actions, buildSnookerActionsForTest(1, 1, []int{20, 35})...)

	actor1, actor2 := calculateSnookerAchievementScoresByActor(actions, rewardMap)
	if actor1 != 10 || actor2 != 0 {
		t.Fatalf("expected actor1 reward 10 and actor2 reward 0, got actor1=%d actor2=%d", actor1, actor2)
	}
}

func TestResolveRankChangeEffectiveAtPrefersEndTime(t *testing.T) {
	matchTime := time.Date(2026, 3, 12, 20, 0, 0, 0, time.UTC)
	endTime := matchTime.Add(45 * time.Minute)
	match := &model.Match{
		MatchTime: matchTime,
		EndTime:   &endTime,
	}

	got := resolveRankChangeEffectiveAt(match)
	if !got.Equal(endTime) {
		t.Fatalf("expected effective time %v, got %v", endTime, got)
	}
}

func TestBuildRankChangeLogIncludesGameType(t *testing.T) {
	effectiveAt := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)

	log := buildRankChangeLog(18, 9, 3, "win", effectiveAt, RankSettlementResult{
		BaseScore:        20,
		AchievementScore: 5,
		FinalChange:      25,
		BeforeScore:      480,
		AfterScore:       505,
		BeforeLevel:      1,
		AfterLevel:       2,
	})

	if log.GameType != 3 {
		t.Fatalf("expected game type 3, got %d", log.GameType)
	}
	if log.MatchId != 18 || log.UserId != 9 || log.Result != "win" {
		t.Fatalf("unexpected rank log payload: %+v", log)
	}
}

func TestBuildRankChangeLogStoresSettlementPolicyRemark(t *testing.T) {
	effectiveAt := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)

	log := buildRankChangeLog(18, 9, 3, "win", effectiveAt, RankSettlementResult{
		BaseScore:              20,
		AchievementScore:       15,
		FinalChange:            10,
		SameOpponentAdjustment: -7,
		DailyCapAdjustment:     -18,
		BeforeScore:            480,
		AfterScore:             490,
		BeforeLevel:            1,
		AfterLevel:             1,
	})

	if !strings.Contains(log.Remark, `"same_opponent_adjustment":-7`) {
		t.Fatalf("expected same opponent adjustment in remark, got %q", log.Remark)
	}
	if !strings.Contains(log.Remark, `"daily_cap_adjustment":-18`) {
		t.Fatalf("expected daily cap adjustment in remark, got %q", log.Remark)
	}
}

func buildSnookerActionsForTest(roundNo int, actor int, scores []int) []model.MatchAction {
	actions := make([]model.MatchAction, 0, len(scores))
	for _, score := range scores {
		actions = append(actions, model.MatchAction{
			RoundNo:     roundNo,
			ActionType:  "score",
			Actor:       actor,
			ScoreChange: score,
		})
	}
	return actions
}
