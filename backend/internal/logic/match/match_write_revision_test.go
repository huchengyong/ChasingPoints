package match

import (
	"testing"

	"chasing_points/internal/model"
)

func TestBuildMatchSyncSnapshotForUserRespectsViewerPerspectiveAndRevision(t *testing.T) {
	opponentID := int64(2002)
	match := &model.Match{
		Id:                        33,
		UserId:                    1001,
		OpponentId:                &opponentID,
		GameType:                  1,
		MyScore:                   6,
		OpponentScore:             4,
		CurrentFrameMyScore:       32,
		CurrentFrameOpponentScore: 18,
		CurrentFrameStarted:       true,
		Status:                    1,
		SyncRevision:              12,
	}
	snookerState := model.SnookerRoundState{
		RedBallCount:           9,
		ClearanceStarted:       true,
		ClearedColors:          []int{2, 3},
		ExpectedClearanceScore: 4,
	}

	player1Snapshot := buildMatchSyncSnapshotForUser(1001, match, 2, snookerState)
	if player1Snapshot.ServerRevision != 12 {
		t.Fatalf("expected player1 snapshot revision 12, got %d", player1Snapshot.ServerRevision)
	}
	if player1Snapshot.MyScore != 6 || player1Snapshot.OpponentScore != 4 {
		t.Fatalf("expected player1 snapshot score 6:4, got %d:%d", player1Snapshot.MyScore, player1Snapshot.OpponentScore)
	}
	if player1Snapshot.CurrentFrameMyScore != 32 || player1Snapshot.CurrentFrameOpponentScore != 18 {
		t.Fatalf("expected player1 current frame 32:18, got %d:%d", player1Snapshot.CurrentFrameMyScore, player1Snapshot.CurrentFrameOpponentScore)
	}
	if player1Snapshot.CurrentRound != 3 || player1Snapshot.TotalRounds != 2 {
		t.Fatalf("expected player1 rounds current=3 total=2, got current=%d total=%d", player1Snapshot.CurrentRound, player1Snapshot.TotalRounds)
	}
	if player1Snapshot.RedBallCount != 9 || !player1Snapshot.SnookerClearanceStarted {
		t.Fatal("expected player1 snapshot to include snooker state")
	}

	player2Snapshot := buildMatchSyncSnapshotForUser(opponentID, match, 2, snookerState)
	if player2Snapshot.MyScore != 4 || player2Snapshot.OpponentScore != 6 {
		t.Fatalf("expected player2 snapshot score 4:6, got %d:%d", player2Snapshot.MyScore, player2Snapshot.OpponentScore)
	}
	if player2Snapshot.CurrentFrameMyScore != 18 || player2Snapshot.CurrentFrameOpponentScore != 32 {
		t.Fatalf("expected player2 current frame 18:32, got %d:%d", player2Snapshot.CurrentFrameMyScore, player2Snapshot.CurrentFrameOpponentScore)
	}
}

func TestBuildMatchSyncSnapshotForUserKeepsFinishedMatchOnCompletedRound(t *testing.T) {
	opponentID := int64(2002)
	match := &model.Match{
		Id:                  34,
		UserId:              1001,
		OpponentId:          &opponentID,
		GameType:            3,
		MyScore:             5,
		OpponentScore:       3,
		CurrentFrameStarted: false,
		Status:              2,
		SyncRevision:        15,
	}

	snapshot := buildMatchSyncSnapshotForUser(1001, match, 5, model.SnookerRoundState{})
	if snapshot.CurrentRound != 5 {
		t.Fatalf("expected completed match current round 5, got %d", snapshot.CurrentRound)
	}
	if snapshot.TotalRounds != 5 {
		t.Fatalf("expected completed match total rounds 5, got %d", snapshot.TotalRounds)
	}
}

func TestBuildMatchWriteScoreViewUsesViewerPerspective(t *testing.T) {
	opponentID := int64(2002)
	match := &model.Match{
		Id:                        35,
		UserId:                    1001,
		OpponentId:                &opponentID,
		MyScore:                   7,
		OpponentScore:             5,
		CurrentFrameMyScore:       42,
		CurrentFrameOpponentScore: 36,
	}

	player1View := buildMatchWriteScoreView(1001, match)
	if player1View.MyScore != 7 || player1View.OpponentScore != 5 {
		t.Fatalf("expected player1 scores 7:5, got %d:%d", player1View.MyScore, player1View.OpponentScore)
	}
	if player1View.CurrentFrameMyScore != 42 || player1View.CurrentFrameOpponentScore != 36 {
		t.Fatalf("expected player1 current frame 42:36, got %d:%d", player1View.CurrentFrameMyScore, player1View.CurrentFrameOpponentScore)
	}

	player2View := buildMatchWriteScoreView(opponentID, match)
	if player2View.MyScore != 5 || player2View.OpponentScore != 7 {
		t.Fatalf("expected player2 scores 5:7, got %d:%d", player2View.MyScore, player2View.OpponentScore)
	}
	if player2View.CurrentFrameMyScore != 36 || player2View.CurrentFrameOpponentScore != 42 {
		t.Fatalf("expected player2 current frame 36:42, got %d:%d", player2View.CurrentFrameMyScore, player2View.CurrentFrameOpponentScore)
	}
}
