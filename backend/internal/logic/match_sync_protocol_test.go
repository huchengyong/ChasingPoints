package logic

import (
	"errors"
	"testing"
)

func TestValidateMatchActionMetaRejectsMissingClientActionID(t *testing.T) {
	err := validateMatchActionMeta(matchActionMeta{
		ClientActionID: "",
		BaseRevision:   4,
	}, 4)
	if err == nil {
		t.Fatal("expected missing client action id to be rejected")
	}
}

func TestValidateMatchActionMetaRejectsStaleBaseRevision(t *testing.T) {
	err := validateMatchActionMeta(matchActionMeta{
		ClientActionID: "action-1",
		BaseRevision:   3,
	}, 4)
	if !errors.Is(err, errRevisionConflict) {
		t.Fatalf("expected revision conflict, got %v", err)
	}
}

func TestValidateMatchActionMetaRejectsFutureBaseRevision(t *testing.T) {
	err := validateMatchActionMeta(matchActionMeta{
		ClientActionID: "action-1",
		BaseRevision:   5,
	}, 4)
	if !errors.Is(err, errRevisionConflict) {
		t.Fatalf("expected revision conflict, got %v", err)
	}
}

func TestBuildActionReplayAckReusesAcceptedMetadata(t *testing.T) {
	ack := buildActionReplayAck("action-1", 9)
	if !ack.Accepted {
		t.Fatal("expected replay ack to stay accepted")
	}
	if ack.ClientActionID != "action-1" {
		t.Fatalf("expected client action id to be preserved, got %q", ack.ClientActionID)
	}
	if ack.ServerRevision != 9 {
		t.Fatalf("expected server revision 9, got %d", ack.ServerRevision)
	}
}
