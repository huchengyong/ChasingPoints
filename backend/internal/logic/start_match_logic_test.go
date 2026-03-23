package logic

import (
	"testing"

	"billiard_master/internal/model"
	"billiard_master/internal/types"
)

func TestEvaluateStartMatchDecisionCreatesWhenNoOngoingMatches(t *testing.T) {
	decision := evaluateStartMatchDecision(100, &types.StartMatchReq{
		GameType:     3,
		OpponentId:   200,
		OpponentName: "球友A",
	}, nil, nil)

	if decision.Action != startMatchActionCreated {
		t.Fatalf("expected action %q, got %q", startMatchActionCreated, decision.Action)
	}
	if decision.BlockReason != "" {
		t.Fatalf("expected empty block reason, got %q", decision.BlockReason)
	}
	if decision.Match != nil {
		t.Fatalf("expected no existing match, got %+v", decision.Match)
	}
}

func TestEvaluateStartMatchDecisionResumesExistingMatchForSameOpponentAndGameType(t *testing.T) {
	existing := &model.Match{
		Id:           88,
		UserId:       100,
		OpponentId:   int64Ptr(200),
		OpponentName: "球友A",
		GameType:     3,
	}

	decision := evaluateStartMatchDecision(100, &types.StartMatchReq{
		GameType:     3,
		OpponentId:   200,
		OpponentName: "球友A",
	}, existing, nil)

	if decision.Action != startMatchActionResumeExisting {
		t.Fatalf("expected action %q, got %q", startMatchActionResumeExisting, decision.Action)
	}
	if decision.Match == nil || decision.Match.Id != 88 {
		t.Fatalf("expected to reuse match 88, got %+v", decision.Match)
	}
}

func TestEvaluateStartMatchDecisionResumesExistingMatchWhenCurrentUserIsStoredAsOpponent(t *testing.T) {
	existing := &model.Match{
		Id:           89,
		UserId:       200,
		OpponentId:   int64Ptr(100),
		OpponentName: "我",
		GameType:     3,
	}

	decision := evaluateStartMatchDecision(100, &types.StartMatchReq{
		GameType:     3,
		OpponentId:   200,
		OpponentName: "球友A",
	}, existing, nil)

	if decision.Action != startMatchActionResumeExisting {
		t.Fatalf("expected action %q, got %q", startMatchActionResumeExisting, decision.Action)
	}
	if decision.Match == nil || decision.Match.Id != 89 {
		t.Fatalf("expected to reuse match 89, got %+v", decision.Match)
	}
}

func TestEvaluateStartMatchDecisionBlocksSelfWhenGameTypeChanges(t *testing.T) {
	existing := &model.Match{
		Id:           90,
		UserId:       100,
		OpponentId:   int64Ptr(200),
		OpponentName: "球友A",
		GameType:     3,
	}

	decision := evaluateStartMatchDecision(100, &types.StartMatchReq{
		GameType:     2,
		OpponentId:   200,
		OpponentName: "球友A",
	}, existing, nil)

	if decision.Action != startMatchActionBlocked {
		t.Fatalf("expected action %q, got %q", startMatchActionBlocked, decision.Action)
	}
	if decision.BlockReason != startMatchBlockReasonSelfOngoing {
		t.Fatalf("expected block reason %q, got %q", startMatchBlockReasonSelfOngoing, decision.BlockReason)
	}
	if decision.Message != "你还有一场中式八球未结束，请先结束之前的对局" {
		t.Fatalf("unexpected block message: %q", decision.Message)
	}
}

func TestEvaluateStartMatchDecisionBlocksSelfWhenOpponentChanges(t *testing.T) {
	existing := &model.Match{
		Id:           91,
		UserId:       100,
		OpponentId:   int64Ptr(201),
		OpponentName: "球友B",
		GameType:     3,
	}

	decision := evaluateStartMatchDecision(100, &types.StartMatchReq{
		GameType:     3,
		OpponentId:   200,
		OpponentName: "球友A",
	}, existing, nil)

	if decision.Action != startMatchActionBlocked {
		t.Fatalf("expected action %q, got %q", startMatchActionBlocked, decision.Action)
	}
	if decision.BlockReason != startMatchBlockReasonSelfOngoing {
		t.Fatalf("expected block reason %q, got %q", startMatchBlockReasonSelfOngoing, decision.BlockReason)
	}
	if decision.Message != "你还有一场中式八球未结束，请先结束之前的对局" {
		t.Fatalf("unexpected block message: %q", decision.Message)
	}
}

func TestEvaluateStartMatchDecisionBlocksWhenOpponentHasAnotherOngoingMatch(t *testing.T) {
	opponentCurrent := &model.Match{
		Id:           92,
		UserId:       200,
		OpponentId:   int64Ptr(300),
		OpponentName: "球友C",
		GameType:     2,
	}

	decision := evaluateStartMatchDecision(100, &types.StartMatchReq{
		GameType:     3,
		OpponentId:   200,
		OpponentName: "球友A",
	}, nil, opponentCurrent)

	if decision.Action != startMatchActionBlocked {
		t.Fatalf("expected action %q, got %q", startMatchActionBlocked, decision.Action)
	}
	if decision.BlockReason != startMatchBlockReasonOpponentOngoing {
		t.Fatalf("expected block reason %q, got %q", startMatchBlockReasonOpponentOngoing, decision.BlockReason)
	}
	if decision.Message != "对手还有未结束的对局，暂时无法开始新的 PK" {
		t.Fatalf("unexpected opponent block message: %q", decision.Message)
	}
}

func TestBuildStartMatchLockUserIDsSortsAndDeduplicatesParticipants(t *testing.T) {
	got := buildStartMatchLockUserIDs(200, 100)

	if len(got) != 2 {
		t.Fatalf("expected 2 ids, got %d", len(got))
	}
	if got[0] != 100 || got[1] != 200 {
		t.Fatalf("expected sorted ids [100 200], got %v", got)
	}
}

func TestBuildStartMatchLockUserIDsSkipsInvalidAndDuplicateIDs(t *testing.T) {
	got := buildStartMatchLockUserIDs(100, 100, 0, -1)

	if len(got) != 1 || got[0] != 100 {
		t.Fatalf("expected deduplicated ids [100], got %v", got)
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}
