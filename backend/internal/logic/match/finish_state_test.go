package match

import (
	"testing"

	"chasing_points/internal/model"
)

func TestRankedFinishCapabilitiesFreezeScoringAndExposeParticipantActions(t *testing.T) {
	opponentID := int64(2002)
	requesterID := int64(1001)
	match := &model.Match{
		UserId:                     requesterID,
		OpponentId:                 &opponentID,
		MatchMode:                  model.MatchModeRanked,
		FinishConfirmationRequired: true,
		FinishState:                model.FinishStatePendingConfirmation,
		FinishRequestedBy:          &requesterID,
		Status:                     1,
	}

	requester := resolveMatchViewerCapabilities(match, requesterID)
	if requester.CanScore || requester.CanUndo || requester.CanConfirmFinish || !requester.CanWithdrawFinish {
		t.Fatalf("unexpected requester capabilities: %+v", requester)
	}

	opponent := resolveMatchViewerCapabilities(match, opponentID)
	if opponent.CanScore || opponent.CanUndo || !opponent.CanConfirmFinish || !opponent.CanDisputeFinish || opponent.CanWithdrawFinish {
		t.Fatalf("unexpected opponent capabilities: %+v", opponent)
	}
}

func TestPracticeAndRankedFinishCapabilitiesUseDifferentEntryPoints(t *testing.T) {
	opponentID := int64(2002)
	practice := &model.Match{UserId: 1001, OpponentId: &opponentID, MatchMode: model.MatchModePractice, Status: 1}
	if capabilities := resolveMatchViewerCapabilities(practice, 1001); !capabilities.CanFinish || capabilities.CanRequestFinish {
		t.Fatalf("unexpected practice capabilities: %+v", capabilities)
	}

	ranked := &model.Match{UserId: 1001, OpponentId: &opponentID, MatchMode: model.MatchModeRanked, FinishConfirmationRequired: true, Status: 1}
	if capabilities := resolveMatchViewerCapabilities(ranked, 1001); capabilities.CanFinish || !capabilities.CanRequestFinish {
		t.Fatalf("unexpected ranked capabilities: %+v", capabilities)
	}

	legacyRanked := &model.Match{UserId: 1001, OpponentId: &opponentID, MatchMode: model.MatchModeRanked, Status: 1}
	if capabilities := resolveMatchViewerCapabilities(legacyRanked, 1001); !capabilities.CanFinish || capabilities.CanRequestFinish {
		t.Fatalf("unexpected legacy ranked capabilities: %+v", capabilities)
	}
}

func TestLegacyMatchWithoutModeKeepsImmediateFinishCompatibility(t *testing.T) {
	opponentID := int64(2002)
	legacy := &model.Match{UserId: 1001, OpponentId: &opponentID, Status: 1}
	if shouldRequestRankedFinish(legacy, 1001) {
		t.Fatal("legacy match without explicit mode should keep immediate finish compatibility")
	}
}

func TestRefereeBoundPendingStateHidesAllPlayerFinishActions(t *testing.T) {
	opponentID := int64(2002)
	requesterID := int64(1001)
	refereeID := int64(3003)
	match := &model.Match{
		UserId: requesterID, OpponentId: &opponentID, RefereeUserId: &refereeID,
		MatchMode: model.MatchModeRanked, FinishConfirmationRequired: true,
		FinishState: model.FinishStatePendingConfirmation, FinishRequestedBy: &requesterID, Status: 1,
	}
	for _, userID := range []int64{requesterID, opponentID} {
		capabilities := resolveMatchViewerCapabilities(match, userID)
		if capabilities.CanConfirmFinish || capabilities.CanDisputeFinish || capabilities.CanWithdrawFinish || capabilities.CanRequestFinish || capabilities.CanFinish {
			t.Fatalf("referee-bound player retained finish capability: user=%d capabilities=%+v", userID, capabilities)
		}
	}
	referee := resolveMatchViewerCapabilities(match, refereeID)
	if !referee.CanFinish || !referee.CanScore || !referee.CanUndo {
		t.Fatalf("referee should keep direct control: %+v", referee)
	}
}
