package ws

import (
	"testing"
	"time"

	"billiard_master/internal/model"
)

func TestBuildMatchSyncDataUsesServerPerspective(t *testing.T) {
	match := &model.Match{
		Id:                        88,
		MyScore:                   7,
		OpponentScore:             5,
		CurrentFrameMyScore:       36,
		CurrentFrameOpponentScore: 29,
		CurrentFrameStarted:       true,
		Status:                    1,
	}

	data := buildMatchSyncData(match, 2, model.SnookerRoundState{}, nil)

	if data.MatchId != 88 {
		t.Fatalf("expected match id 88, got %d", data.MatchId)
	}
	if data.Player1Score != 7 {
		t.Fatalf("expected player1 score 7, got %d", data.Player1Score)
	}
	if data.Player2Score != 5 {
		t.Fatalf("expected player2 score 5, got %d", data.Player2Score)
	}
	if data.CurrentFramePlayer1Score != 36 {
		t.Fatalf("expected current frame player1 score 36, got %d", data.CurrentFramePlayer1Score)
	}
	if data.CurrentFramePlayer2Score != 29 {
		t.Fatalf("expected current frame player2 score 29, got %d", data.CurrentFramePlayer2Score)
	}
	if !data.CurrentFrameStarted {
		t.Fatalf("expected current frame started flag true")
	}
	if data.CurrentRound != 3 {
		t.Fatalf("expected current round 3, got %d", data.CurrentRound)
	}
	if data.TotalRounds != 2 {
		t.Fatalf("expected total rounds 2, got %d", data.TotalRounds)
	}
}

func TestBuildMatchSyncDataFiltersUnfinishedRounds(t *testing.T) {
	now := time.Now()
	data := buildMatchSyncData(&model.Match{Id: 1, Status: 1}, 1, model.SnookerRoundState{}, []model.MatchRound{
		{
			Id:            1,
			RoundNo:       1,
			MyScore:       3,
			OpponentScore: 1,
			Winner:        intPtr(1),
			WinType:       "normal",
			CreatedAt:     now,
		},
		{
			Id:            2,
			RoundNo:       2,
			MyScore:       3,
			OpponentScore: 1,
			Winner:        nil,
			WinType:       "start",
			CreatedAt:     now,
		},
	})

	if len(data.Rounds) != 1 {
		t.Fatalf("expected 1 finished round, got %d", len(data.Rounds))
	}
	if data.Rounds[0].RoundNumber != 1 {
		t.Fatalf("expected round number 1, got %d", data.Rounds[0].RoundNumber)
	}
	if data.Rounds[0].Winner != 1 {
		t.Fatalf("expected winner 1, got %d", data.Rounds[0].Winner)
	}
}

func intPtr(v int) *int {
	return &v
}
