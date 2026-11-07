package model

import (
	"strings"
	"testing"
)

func newStrokeEvent(state SnookerRoundState, actor int, outcome string) SnookerEvent {
	return SnookerEvent{
		Version: SnookerEventVersion,
		Kind:    SnookerEventKindStroke,
		Actor:   actor,
		VisitNo: state.VisitNo,
		Outcome: outcome,
	}
}

func applyStrokeForTest(t *testing.T, state SnookerRoundState, event SnookerEvent) SnookerRoundState {
	t.Helper()
	transition, err := ApplySnookerEvent(state, event)
	if err != nil {
		t.Fatalf("apply stroke: %v", err)
	}
	return transition.State
}

func TestSnookerV2FormatAndRoundInitialization(t *testing.T) {
	if err := ValidateSnookerMatchFormat(7, 2); err != nil {
		t.Fatalf("valid format rejected: %v", err)
	}
	for _, input := range []struct {
		bestOf  int
		starter int
	}{{0, 1}, {4, 1}, {7, 0}, {7, 3}} {
		if err := ValidateSnookerMatchFormat(input.bestOf, input.starter); err == nil {
			t.Fatalf("invalid format accepted: %+v", input)
		}
	}
	if got := SnookerStartingActor(2, 1); got != 2 {
		t.Fatalf("round 1 starter=%d", got)
	}
	if got := SnookerStartingActor(2, 2); got != 1 {
		t.Fatalf("round 2 starter=%d", got)
	}
	if got := SnookerStartingActor(2, 3); got != 2 {
		t.Fatalf("round 3 starter=%d", got)
	}

	state, err := NewSnookerRoundStateV2(1, 2)
	if err != nil {
		t.Fatalf("init state: %v", err)
	}
	if state.Phase != SnookerPhaseReds || state.BallOn != SnookerBallRed || state.Striker != 2 ||
		state.VisitNo != 1 || state.RedsRemaining != 15 || state.RedBallCount != 0 {
		t.Fatalf("unexpected initial state: %+v", state)
	}
}

func TestSnookerV2RejectsBlackAtOpening(t *testing.T) {
	state, _ := NewSnookerRoundStateV2(1, 1)
	black := newStrokeEvent(state, 1, SnookerOutcomePot)
	black.BallOnValue = 7
	black.BallOnPotted = true
	if _, err := ApplySnookerEvent(state, black); err == nil {
		t.Fatal("expected opening black rejection")
	}
}

func TestSnookerV2RedColorAndVisitTransitions(t *testing.T) {
	state, _ := NewSnookerRoundStateV2(1, 1)
	red := newStrokeEvent(state, 1, SnookerOutcomePot)
	red.PottedReds = 2
	state = applyStrokeForTest(t, state, red)
	if state.Player1Score != 2 || state.RedsRemaining != 13 || state.BallOn != SnookerBallColorChoice || state.CurrentBreak != 2 {
		t.Fatalf("unexpected multi-red state: %+v", state)
	}

	black := newStrokeEvent(state, 1, SnookerOutcomePot)
	black.BallOnValue = 7
	black.BallOnPotted = true
	state = applyStrokeForTest(t, state, black)
	if state.Player1Score != 9 || state.BallOn != SnookerBallRed || state.CurrentBreak != 9 {
		t.Fatalf("unexpected color state: %+v", state)
	}

	miss := newStrokeEvent(state, 1, SnookerOutcomeNoScore)
	state = applyStrokeForTest(t, state, miss)
	if state.Striker != 2 || state.VisitNo != 2 || state.CurrentBreak != 0 || state.Player1Score != 9 {
		t.Fatalf("unexpected visit switch: %+v", state)
	}
}

func TestSnookerV2MissingLastRedKeepsRedPhase(t *testing.T) {
	state, _ := NewSnookerRoundStateV2(1, 1)
	state.RedsRemaining = 1
	syncSnookerLegacyFields(&state)
	miss := newStrokeEvent(state, 1, SnookerOutcomeNoScore)
	state = applyStrokeForTest(t, state, miss)
	if state.Phase != SnookerPhaseReds || state.BallOn != SnookerBallRed || state.RedsRemaining != 1 || state.Striker != 2 {
		t.Fatalf("last red miss should keep red on table: %+v", state)
	}
}

func TestSnookerV2LastRedColorMissStartsYellow(t *testing.T) {
	state, _ := NewSnookerRoundStateV2(1, 1)
	state.RedsRemaining = 1
	syncSnookerLegacyFields(&state)
	red := newStrokeEvent(state, 1, SnookerOutcomePot)
	red.PottedReds = 1
	state = applyStrokeForTest(t, state, red)
	if state.Phase != SnookerPhaseColorAfterRed || state.RedsRemaining != 0 {
		t.Fatalf("last red transition failed: %+v", state)
	}

	miss := newStrokeEvent(state, 1, SnookerOutcomeNoScore)
	miss.BallOnValue = 7
	state = applyStrokeForTest(t, state, miss)
	if state.Phase != SnookerPhaseColors || state.BallOn != SnookerBallYellow || state.Striker != 2 {
		t.Fatalf("color after last red miss failed: %+v", state)
	}
}

func TestSnookerV2RejectsIllegalActorAndClearanceBall(t *testing.T) {
	state, _ := NewSnookerRoundStateV2(1, 1)
	wrongActor := newStrokeEvent(state, 2, SnookerOutcomePot)
	wrongActor.PottedReds = 1
	if _, err := ApplySnookerEvent(state, wrongActor); err == nil {
		t.Fatal("expected non-striker rejection")
	}

	state.RedsRemaining = 0
	state.Phase = SnookerPhaseColors
	state.BallOn = SnookerBallYellow
	syncSnookerLegacyFields(&state)
	wrongColor := newStrokeEvent(state, 1, SnookerOutcomePot)
	wrongColor.BallOnValue = 3
	wrongColor.BallOnPotted = true
	if _, err := ApplySnookerEvent(state, wrongColor); err == nil {
		t.Fatal("expected clearance order rejection")
	}
}

func TestSnookerV2Legal147(t *testing.T) {
	state, _ := NewSnookerRoundStateV2(1, 1)
	for redNo := 0; redNo < 15; redNo++ {
		red := newStrokeEvent(state, 1, SnookerOutcomePot)
		red.PottedReds = 1
		state = applyStrokeForTest(t, state, red)

		black := newStrokeEvent(state, 1, SnookerOutcomePot)
		black.BallOnValue = 7
		black.BallOnPotted = true
		state = applyStrokeForTest(t, state, black)
	}
	for value := 2; value <= 7; value++ {
		color := newStrokeEvent(state, 1, SnookerOutcomePot)
		color.BallOnValue = value
		color.BallOnPotted = true
		state = applyStrokeForTest(t, state, color)
	}
	if state.Player1Score != 147 || !state.FrameEnded || state.FrameWinner != 1 || state.FrameEndReason != SnookerFrameEndClearance {
		t.Fatalf("unexpected legal 147 result: %+v", state)
	}
}

func TestSnookerV2RejectsInconsistentNoScoreFields(t *testing.T) {
	state, _ := NewSnookerRoundStateV2(1, 1)
	wrongTarget := newStrokeEvent(state, 1, SnookerOutcomeNoScore)
	wrongTarget.BallOnValue = 7
	if _, err := ApplySnookerEvent(state, wrongTarget); err == nil {
		t.Fatal("expected mismatched red target rejection")
	}

	state.Phase = SnookerPhaseColorAfterRed
	state.BallOn = SnookerBallColorChoice
	missingColor := newStrokeEvent(state, 1, SnookerOutcomeNoScore)
	if _, err := ApplySnookerEvent(state, missingColor); err == nil {
		t.Fatal("expected missing nominated color rejection")
	}

	state, _ = NewSnookerRoundStateV2(1, 1)
	invalidCueBall := newStrokeEvent(state, 1, SnookerOutcomeNoScore)
	invalidCueBall.CueBallInHand = true
	if _, err := ApplySnookerEvent(state, invalidCueBall); err == nil {
		t.Fatal("expected no-score cue-ball field rejection")
	}
}

func TestSnookerV2ClearanceOrder(t *testing.T) {
	state, _ := NewSnookerRoundStateV2(1, 1)
	state.RedsRemaining = 0
	state.Phase = SnookerPhaseColors
	state.BallOn = SnookerBallYellow
	syncSnookerLegacyFields(&state)
	for value := 2; value <= 6; value++ {
		event := newStrokeEvent(state, 1, SnookerOutcomePot)
		event.BallOnValue = value
		event.BallOnPotted = true
		state = applyStrokeForTest(t, state, event)
	}
	if state.BallOn != SnookerBallBlack || len(state.ClearedColors) != 5 || state.Player1Score != 20 {
		t.Fatalf("unexpected clearance state: %+v", state)
	}
}

func TestSnookerV2FoulPenaltyAndRemovedReds(t *testing.T) {
	state, _ := NewSnookerRoundStateV2(1, 1)
	invalid := newStrokeEvent(state, 1, SnookerOutcomeFoul)
	invalid.Penalty = 3
	invalid.FoulResolution = SnookerFoulIncomingPlays
	if _, err := ApplySnookerEvent(state, invalid); err == nil {
		t.Fatal("expected penalty below four rejection")
	}

	foul := newStrokeEvent(state, 1, SnookerOutcomeFoul)
	foul.Penalty = 4
	foul.RedsRemoved = 2
	foul.FoulResolution = SnookerFoulIncomingPlays
	state = applyStrokeForTest(t, state, foul)
	if state.Player2Score != 4 || state.Player1Score != 0 || state.RedsRemaining != 13 || state.Striker != 2 || state.VisitNo != 2 {
		t.Fatalf("unexpected foul result: %+v", state)
	}
}

func TestSnookerV2FoulReplayRestoresBallsAndKeepsOffender(t *testing.T) {
	state, _ := NewSnookerRoundStateV2(1, 1)
	foul := newStrokeEvent(state, 1, SnookerOutcomeFoul)
	foul.Penalty = 4
	foul.RedsRemoved = 2
	foul.FoulResolution = SnookerFoulOffenderReplaysOriginal
	state = applyStrokeForTest(t, state, foul)
	if state.RedsRemaining != 15 || state.Striker != 1 || state.Player2Score != 4 || state.VisitNo != 2 {
		t.Fatalf("unexpected replay state: %+v", state)
	}
}

func TestSnookerV2FreeBallAsRedAndWithRealReds(t *testing.T) {
	state, _ := NewSnookerRoundStateV2(1, 1)
	state.FreeBallAvailable = true
	freeRed := newStrokeEvent(state, 1, SnookerOutcomePot)
	freeRed.FreeBallValue = 2
	freeRed.FreeBallPotted = true
	state = applyStrokeForTest(t, state, freeRed)
	if state.Player1Score != 1 || state.RedsRemaining != 15 || state.BallOn != SnookerBallColorChoice {
		t.Fatalf("unexpected free ball as red: %+v", state)
	}

	state, _ = NewSnookerRoundStateV2(1, 1)
	state.FreeBallAvailable = true
	freeAndReds := newStrokeEvent(state, 1, SnookerOutcomePot)
	freeAndReds.FreeBallValue = 7
	freeAndReds.FreeBallPotted = true
	freeAndReds.PottedReds = 2
	state = applyStrokeForTest(t, state, freeAndReds)
	if state.Player1Score != 3 || state.RedsRemaining != 13 || state.CurrentBreak != 3 {
		t.Fatalf("unexpected free ball plus reds: %+v", state)
	}
}

func TestSnookerV2FreeBallCanLeaveNomineeUpAndPotActualBallOn(t *testing.T) {
	state, _ := NewSnookerRoundStateV2(1, 1)
	state.FreeBallAvailable = true
	red := newStrokeEvent(state, 1, SnookerOutcomePot)
	red.FreeBallValue = 2
	red.PottedReds = 1
	state = applyStrokeForTest(t, state, red)
	if state.Player1Score != 1 || state.RedsRemaining != 14 || state.BallOn != SnookerBallColorChoice {
		t.Fatalf("unexpected real red after free-ball nomination: %+v", state)
	}

	state, _ = NewSnookerRoundStateV2(1, 1)
	state.RedsRemaining = 0
	state.Phase = SnookerPhaseColors
	state.BallOn = SnookerBallYellow
	state.FreeBallAvailable = true
	syncSnookerLegacyFields(&state)
	yellow := newStrokeEvent(state, 1, SnookerOutcomePot)
	yellow.BallOnValue = 2
	yellow.BallOnPotted = true
	yellow.FreeBallValue = 3
	state = applyStrokeForTest(t, state, yellow)
	if state.Player1Score != 2 || state.BallOn != SnookerBallGreen {
		t.Fatalf("unexpected actual color after free-ball nomination: %+v", state)
	}

	state, _ = NewSnookerRoundStateV2(1, 1)
	state.FreeBallAvailable = true
	noScore := newStrokeEvent(state, 1, SnookerOutcomeNoScore)
	noScore.FreeBallValue = 2
	state = applyStrokeForTest(t, state, noScore)
	if state.Striker != 2 || state.FreeBallAvailable {
		t.Fatalf("free-ball no-score should end visit: %+v", state)
	}
}

func TestSnookerV2ClearanceFreeBallDoesNotRemoveTarget(t *testing.T) {
	state, _ := NewSnookerRoundStateV2(1, 1)
	state.RedsRemaining = 0
	state.Phase = SnookerPhaseColors
	state.BallOn = SnookerBallYellow
	state.FreeBallAvailable = true
	syncSnookerLegacyFields(&state)
	event := newStrokeEvent(state, 1, SnookerOutcomePot)
	event.BallOnValue = 2
	event.FreeBallValue = 3
	event.FreeBallPotted = true
	state = applyStrokeForTest(t, state, event)
	if state.Player1Score != 2 || state.BallOn != SnookerBallYellow || len(state.ClearedColors) != 0 {
		t.Fatalf("free ball should not remove actual yellow: %+v", state)
	}

	state.FreeBallAvailable = true
	event = newStrokeEvent(state, 1, SnookerOutcomePot)
	event.BallOnValue = 2
	event.FreeBallValue = 3
	event.FreeBallPotted = true
	event.BallOnPotted = true
	state = applyStrokeForTest(t, state, event)
	if state.Player1Score != 4 || state.BallOn != SnookerBallGreen || len(state.ClearedColors) != 1 {
		t.Fatalf("actual yellow should advance clearance: %+v", state)
	}
}

func TestSnookerV2MissSequenceWarnsThenAwardsFrame(t *testing.T) {
	state, _ := NewSnookerRoundStateV2(1, 1)
	for attempt := 1; attempt <= 3; attempt++ {
		event := newStrokeEvent(state, 1, SnookerOutcomeFoul)
		event.Penalty = 4
		event.FoulAndMiss = true
		event.MissSequenceEligible = true
		event.FoulResolution = SnookerFoulOffenderReplaysOriginal
		state = applyStrokeForTest(t, state, event)
		if attempt == 2 && !state.MissWarningActive {
			t.Fatal("expected warning after second eligible miss")
		}
	}
	if !state.FrameEnded || state.FrameWinner != 2 || state.FrameEndReason != SnookerFrameEndRepeatedMiss {
		t.Fatalf("expected frame award after third miss: %+v", state)
	}
}

func finalBlackState(player1, player2, striker int) SnookerRoundState {
	state, _ := NewSnookerRoundStateV2(1, striker)
	state.RedsRemaining = 0
	state.Phase = SnookerPhaseColors
	state.BallOn = SnookerBallBlack
	state.ClearedColors = []int{2, 3, 4, 5, 6}
	state.Player1Score = player1
	state.Player2Score = player2
	syncSnookerLegacyFields(&state)
	return state
}

func TestSnookerV2FinalBlackAndRespottedBlack(t *testing.T) {
	state := finalBlackState(50, 43, 2)
	black := newStrokeEvent(state, 2, SnookerOutcomePot)
	black.BallOnValue = 7
	black.BallOnPotted = true
	state = applyStrokeForTest(t, state, black)
	if state.Phase != SnookerPhaseRespottedBlackPending || state.FrameEnded {
		t.Fatalf("expected respotted black pending: %+v", state)
	}

	start := SnookerEvent{Version: SnookerEventVersion, Kind: SnookerEventKindFrameAction, Actor: 1, FrameAction: SnookerFrameActionStartRespottedBlack}
	transition, err := ApplySnookerEvent(state, start)
	if err != nil {
		t.Fatalf("start respotted black: %v", err)
	}
	state = transition.State
	if state.Phase != SnookerPhaseRespottedBlack || state.Striker != 1 || !state.CueBallInHand {
		t.Fatalf("unexpected respotted black state: %+v", state)
	}

	foul := newStrokeEvent(state, 1, SnookerOutcomeFoul)
	foul.Penalty = 7
	state = applyStrokeForTest(t, state, foul)
	if !state.FrameEnded || state.FrameWinner != 2 || state.FrameEndReason != SnookerFrameEndRespottedBlack {
		t.Fatalf("respotted black foul should end frame: %+v", state)
	}
}

func TestSnookerV2RespottedBlackWaitsForConcessionDecision(t *testing.T) {
	state := finalBlackState(50, 43, 2)
	black := newStrokeEvent(state, 2, SnookerOutcomePot)
	black.BallOnValue = 7
	black.BallOnPotted = true
	state = applyStrokeForTest(t, state, black)

	offer := SnookerEvent{
		Version: SnookerEventVersion, Kind: SnookerEventKindFrameAction,
		Actor: 1, FrameAction: SnookerFrameActionOfferConcession, Scope: SnookerConcessionScopeMatch,
	}
	transition, err := ApplySnookerEvent(state, offer)
	if err != nil {
		t.Fatalf("offer match concession: %v", err)
	}
	start := SnookerEvent{Version: SnookerEventVersion, Kind: SnookerEventKindFrameAction, Actor: 1, FrameAction: SnookerFrameActionStartRespottedBlack}
	if _, err := ApplySnookerEvent(transition.State, start); err == nil {
		t.Fatal("expected respotted black to wait for concession decision")
	}
}

func TestSnookerV2FinalBlackWithoutTieEndsFrame(t *testing.T) {
	state := finalBlackState(60, 40, 1)
	black := newStrokeEvent(state, 1, SnookerOutcomePot)
	black.BallOnValue = 7
	black.BallOnPotted = true
	state = applyStrokeForTest(t, state, black)
	if !state.FrameEnded || state.FrameWinner != 1 || state.FrameEndReason != SnookerFrameEndClearance {
		t.Fatalf("expected normal frame completion: %+v", state)
	}
}

func TestSnookerV2RemainingPointsAndConcession(t *testing.T) {
	state := finalBlackState(20, 40, 1)
	state.BallOn = SnookerBallPink
	state.ClearedColors = []int{2, 3, 4, 5}
	syncSnookerLegacyFields(&state)
	if got := SnookerRemainingPoints(state); got != 13 {
		t.Fatalf("remaining points=%d", got)
	}
	if !CanOfferSnookerFrameConcession(state, 1) {
		t.Fatal("trailing player should require penalty points")
	}

	offer := SnookerEvent{Version: SnookerEventVersion, Kind: SnookerEventKindFrameAction, Actor: 1, FrameAction: SnookerFrameActionOfferConcession, Scope: SnookerConcessionScopeFrame}
	transition, err := ApplySnookerEvent(state, offer)
	if err != nil {
		t.Fatalf("offer concession: %v", err)
	}
	accept := SnookerEvent{Version: SnookerEventVersion, Kind: SnookerEventKindFrameAction, Actor: 2, FrameAction: SnookerFrameActionAcceptConcession}
	transition, err = ApplySnookerEvent(transition.State, accept)
	if err != nil {
		t.Fatalf("accept concession: %v", err)
	}
	if !transition.State.FrameEnded || transition.State.FrameWinner != 2 || transition.State.FrameEndReason != SnookerFrameEndConcession {
		t.Fatalf("unexpected concession result: %+v", transition.State)
	}
}

func TestSnookerV2RejectsEarlyConcession(t *testing.T) {
	state, _ := NewSnookerRoundStateV2(1, 1)
	state.Player1Score = 0
	state.Player2Score = 10
	offer := SnookerEvent{Version: SnookerEventVersion, Kind: SnookerEventKindFrameAction, Actor: 1, FrameAction: SnookerFrameActionOfferConcession, Scope: SnookerConcessionScopeFrame}
	if _, err := ApplySnookerEvent(state, offer); err == nil {
		t.Fatal("expected early concession rejection")
	}
}

func TestReplaySnookerRoundV2RejectsMixedLegacyOrUnknownActions(t *testing.T) {
	for _, actionType := range []string{"score", "foul", "unexpected"} {
		actions := []MatchAction{{Id: 1, RoundNo: 1, ActionType: actionType, Actor: 1, ScoreChange: 1}}
		if _, err := ReplaySnookerRoundV2(actions, 1, 1); err == nil {
			t.Fatalf("expected %s action rejection", actionType)
		}
	}
	invalidRoundStart := []MatchAction{{Id: 2, RoundNo: 1, ActionType: "round_start", Actor: 1}}
	if _, err := ReplaySnookerRoundV2(invalidRoundStart, 1, 1); err == nil {
		t.Fatal("expected invalid round_start rejection")
	}
}

func TestSnookerFrameActionRejectsContradictoryFields(t *testing.T) {
	state, _ := NewSnookerRoundStateV2(1, 1)
	award := SnookerEvent{
		Version: SnookerEventVersion, Kind: SnookerEventKindFrameAction,
		Actor: 2, FrameAction: SnookerFrameActionAwardFrame, Winner: 1, Reason: "裁判判局",
	}
	if _, err := ApplySnookerEvent(state, award); err == nil {
		t.Fatal("expected award actor and winner mismatch rejection")
	}
	start := SnookerEvent{
		Version: SnookerEventVersion, Kind: SnookerEventKindFrameAction,
		Actor: 1, FrameAction: SnookerFrameActionStartRespottedBlack, Winner: 1,
	}
	if _, err := EncodeSnookerEvent(start); err == nil {
		t.Fatal("expected respotted black extra winner rejection")
	}

	state.PendingConcessionActor = 1
	state.PendingConcessionScope = SnookerConcessionScopeMatch
	accept := SnookerEvent{
		Version: SnookerEventVersion, Kind: SnookerEventKindFrameAction,
		Actor: 2, FrameAction: SnookerFrameActionAcceptConcession, Scope: SnookerConcessionScopeFrame,
	}
	if _, err := ApplySnookerEvent(state, accept); err == nil {
		t.Fatal("expected concession scope mismatch rejection")
	}
}

func TestSnookerEventCodecRejectsUnknownOrIncompletePayload(t *testing.T) {
	state, _ := NewSnookerRoundStateV2(1, 1)
	event := newStrokeEvent(state, 1, SnookerOutcomeNoScore)
	raw, err := EncodeSnookerEvent(event)
	if err != nil {
		t.Fatalf("encode event: %v", err)
	}
	decoded, err := DecodeSnookerEvent(&raw)
	if err != nil || decoded.Outcome != SnookerOutcomeNoScore {
		t.Fatalf("decode event: event=%+v err=%v", decoded, err)
	}

	unknownVersion := strings.Replace(raw, `"version":2`, `"version":3`, 1)
	if _, err := DecodeSnookerEvent(&unknownVersion); err == nil {
		t.Fatal("expected unknown version rejection")
	}
	unknownField := strings.TrimSuffix(raw, "}") + `,"unexpected":true}`
	if _, err := DecodeSnookerEvent(&unknownField); err == nil {
		t.Fatal("expected unknown field rejection")
	}
}
