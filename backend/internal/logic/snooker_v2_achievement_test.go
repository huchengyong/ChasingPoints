package logic

import (
	"testing"

	"chasing_points/internal/model"
)

func snookerV2ActionForStats(t *testing.T, roundNo, actor, visit, score int, outcome string) model.MatchAction {
	t.Helper()
	event := model.SnookerEvent{
		Version: model.SnookerEventVersion,
		Kind:    model.SnookerEventKindStroke,
		Actor:   actor,
		VisitNo: visit,
		Outcome: outcome,
	}
	if outcome == model.SnookerOutcomeFoul {
		event.Penalty = score
		event.FoulResolution = model.SnookerFoulIncomingPlays
	}
	raw, err := model.EncodeSnookerEvent(event)
	if err != nil {
		t.Fatalf("encode event: %v", err)
	}
	return model.MatchAction{
		RoundNo:     roundNo,
		Actor:       actor,
		ActionType:  model.MatchActionTypeSnookerStroke,
		ScoreChange: score,
		ExtraData:   &raw,
	}
}

func TestSnookerV2BreaksUseVisitBoundaries(t *testing.T) {
	actions := []model.MatchAction{
		snookerV2ActionForStats(t, 1, 1, 1, 30, model.SnookerOutcomePot),
		snookerV2ActionForStats(t, 1, 1, 1, 0, model.SnookerOutcomeNoScore),
		snookerV2ActionForStats(t, 1, 2, 2, 0, model.SnookerOutcomeNoScore),
		snookerV2ActionForStats(t, 1, 1, 3, 30, model.SnookerOutcomePot),
	}
	stats := calculateSnookerBreakStats(actions, 1)
	if stats.Highest != 30 || stats.FiftyPlus != 0 || stats.Centuries != 0 {
		t.Fatalf("visits were merged: %+v", stats)
	}
}

func TestSnookerV2BreakRewardsDistinguish147And155(t *testing.T) {
	actions := []model.MatchAction{
		snookerV2ActionForStats(t, 1, 1, 1, 147, model.SnookerOutcomePot),
		snookerV2ActionForStats(t, 2, 2, 1, 155, model.SnookerOutcomePot),
		snookerV2ActionForStats(t, 2, 1, 2, 7, model.SnookerOutcomeFoul),
	}
	rewards := map[string]int{"break_50": 5, "break_100": 10, "break_147": 50}
	player1, player2, err := calculateSnookerAchievementScoresByActorStrict(actions, rewards)
	if err != nil {
		t.Fatalf("calculate rewards: %v", err)
	}
	if player1 != 50 || player2 != 10 {
		t.Fatalf("unexpected rewards: player1=%d player2=%d", player1, player2)
	}
	counts, err := CalculateSnookerBreakAchievementCounts(actions)
	if err != nil {
		t.Fatalf("calculate counts: %v", err)
	}
	if counts[1].FiftyPlus != 1 || counts[1].Centuries != 1 || counts[1].Break147 != 1 {
		t.Fatalf("unexpected actor1 counts: %+v", counts[1])
	}
	if counts[2].FiftyPlus != 1 || counts[2].Centuries != 1 || counts[2].Break147 != 0 {
		t.Fatalf("unexpected actor2 counts: %+v", counts[2])
	}
	if got := calculateSnookerHighestBreak(actions, 2); got != 155 {
		t.Fatalf("expected displayed 155, got %d", got)
	}
}

func TestSnookerV2UnknownActionBlocksStrictSettlement(t *testing.T) {
	actions := []model.MatchAction{
		snookerV2ActionForStats(t, 1, 1, 1, 1, model.SnookerOutcomePot),
		{Id: 98, RoundNo: 1, Actor: 0, ActionType: "unexpected"},
	}
	if _, _, err := calculateSnookerAchievementScoresByActorStrict(actions, map[string]int{}); err == nil {
		t.Fatal("expected unknown action to block settlement")
	}
}

func TestSnookerV2InvalidEventBlocksStrictSettlement(t *testing.T) {
	raw := `{"version":3,"kind":"stroke","actor":1,"visit_no":1,"outcome":"pot"}`
	actions := []model.MatchAction{{
		Id:          99,
		RoundNo:     1,
		Actor:       1,
		ActionType:  model.MatchActionTypeSnookerStroke,
		ScoreChange: 7,
		ExtraData:   &raw,
	}}
	if _, _, err := calculateSnookerAchievementScoresByActorStrict(actions, map[string]int{}); err == nil {
		t.Fatal("expected malformed event to block settlement")
	}
}
