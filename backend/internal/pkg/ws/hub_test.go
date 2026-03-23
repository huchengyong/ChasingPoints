package ws

import (
	"testing"

	"billiard_master/internal/model"
)

func TestBuildMatchSyncDataIncludesServerRevision(t *testing.T) {
	match := &model.Match{
		Id:                        36,
		MyScore:                   4,
		OpponentScore:             2,
		CurrentFrameMyScore:       18,
		CurrentFrameOpponentScore: 7,
		CurrentFrameStarted:       true,
		Status:                    1,
		SyncRevision:              9,
	}

	data := buildMatchSyncData(match, 1, model.SnookerRoundState{}, nil)
	if data.ServerRevision != 9 {
		t.Fatalf("expected sync data server revision 9, got %d", data.ServerRevision)
	}
}
