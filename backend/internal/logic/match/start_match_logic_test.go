package match

import (
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
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

func TestValidateStartMatchReqRejectsMissingOpponentID(t *testing.T) {
	message := validateStartMatchReq(100, &types.StartMatchReq{
		GameType:     3,
		OpponentId:   0,
		OpponentName: "球友A",
	})

	if message != "请选择有效的平台对手" {
		t.Fatalf("unexpected validation message: %q", message)
	}
}

func TestValidateStartMatchReqRejectsSelfMatch(t *testing.T) {
	message := validateStartMatchReq(100, &types.StartMatchReq{
		GameType:     3,
		OpponentId:   100,
		OpponentName: "自己",
	})

	if message != "不能和自己发起 PK" {
		t.Fatalf("unexpected validation message: %q", message)
	}
}

func TestValidateStartMatchReqNegotiatesSnookerRulesVersion(t *testing.T) {
	legacy := &types.StartMatchReq{GameType: 1, OpponentId: 200}
	if message := validateStartMatchReq(100, legacy); message != "" {
		t.Fatalf("legacy client should remain compatible: %q", message)
	}
	if legacy.SnookerRulesVersion != model.SnookerRulesVersionLegacy || legacy.BestOfFrames != 0 || legacy.SnookerFormat != model.SnookerFormatLegacy || legacy.SnookerTargetWins != 0 || legacy.StartingActor != 0 {
		t.Fatalf("unexpected legacy negotiation: %+v", legacy)
	}

	version2 := &types.StartMatchReq{
		GameType: 1, OpponentId: 200, SnookerRulesVersion: model.SnookerRulesVersionWPBSA,
		BestOfFrames: 7, StartingActor: 2,
	}
	if message := validateStartMatchReq(100, version2); message != "" {
		t.Fatalf("valid version 2 rejected: %q", message)
	}
	if version2.SnookerFormat != model.SnookerFormatRaceTo || version2.SnookerTargetWins != 4 {
		t.Fatalf("legacy best-of should map to race-to threshold: %+v", version2)
	}
	free := &types.StartMatchReq{
		GameType: 1, OpponentId: 200, SnookerRulesVersion: model.SnookerRulesVersionWPBSA,
		SnookerFormat: model.SnookerFormatFree,
	}
	if message := validateStartMatchReq(100, free); message != "" || free.SnookerTargetWins != 0 || free.StartingActor != 1 {
		t.Fatalf("valid free format rejected or not normalized: req=%+v message=%q", free, message)
	}
	raceTo := &types.StartMatchReq{
		GameType: 1, OpponentId: 200, SnookerRulesVersion: model.SnookerRulesVersionWPBSA,
		SnookerFormat: model.SnookerFormatRaceTo, SnookerTargetWins: 10,
	}
	if message := validateStartMatchReq(100, raceTo); message != "" || raceTo.StartingActor != 1 {
		t.Fatalf("valid race-to format rejected or not normalized: req=%+v message=%q", raceTo, message)
	}
	if message := validateStartMatchReq(100, &types.StartMatchReq{
		GameType: 1, OpponentId: 200, SnookerRulesVersion: model.SnookerRulesVersionWPBSA,
		SnookerFormat: model.SnookerFormatFree, SnookerTargetWins: 1,
	}); message != "请选择有效的斯诺克赛制" {
		t.Fatalf("unexpected invalid free format message: %q", message)
	}
	if message := validateStartMatchReq(100, &types.StartMatchReq{
		GameType: 1, OpponentId: 200, SnookerRulesVersion: model.SnookerRulesVersionWPBSA,
		SnookerFormat: model.SnookerFormatRaceTo, SnookerTargetWins: 26,
	}); message != "请选择有效的斯诺克赛制" {
		t.Fatalf("unexpected invalid race-to format message: %q", message)
	}
	if message := validateStartMatchReq(100, &types.StartMatchReq{
		GameType: 1, OpponentId: 200, SnookerRulesVersion: model.SnookerRulesVersionWPBSA,
	}); message != "斯诺克总局数必须为正奇数" {
		t.Fatalf("unexpected invalid format message: %q", message)
	}
	if message := validateStartMatchReq(100, &types.StartMatchReq{
		GameType: 1, OpponentId: 200, SnookerRulesVersion: 3,
	}); message != "不支持的斯诺克规则版本" {
		t.Fatalf("unexpected unsupported version message: %q", message)
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
