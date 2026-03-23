package logic

import (
	"testing"

	"chasing_points/internal/model"
)

func TestApplySnookerScoreOnlyUpdatesCurrentFrameScore(t *testing.T) {
	match := &model.Match{
		GameType:                  1,
		MyScore:                   2,
		OpponentScore:             1,
		CurrentFrameMyScore:       28,
		CurrentFrameOpponentScore: 14,
		CurrentFrameStarted:       true,
	}

	applySnookerScore(match, 1, 7)
	applySnookerFoul(match, 1, 4)

	if match.MyScore != 2 || match.OpponentScore != 1 {
		t.Fatalf("expected frame win score unchanged, got %d:%d", match.MyScore, match.OpponentScore)
	}
	if match.CurrentFrameMyScore != 35 || match.CurrentFrameOpponentScore != 18 {
		t.Fatalf("expected current frame score 35:18, got %d:%d", match.CurrentFrameMyScore, match.CurrentFrameOpponentScore)
	}
}

func TestFinalizeSnookerFramePromotesFrameWinAndClearsCurrentFrame(t *testing.T) {
	match := &model.Match{
		GameType:                  1,
		MyScore:                   3,
		OpponentScore:             2,
		CurrentFrameMyScore:       64,
		CurrentFrameOpponentScore: 37,
		CurrentFrameStarted:       true,
	}

	round := finalizeSnookerFrame(match, 1)

	if round.MyScore != 64 || round.OpponentScore != 37 {
		t.Fatalf("expected finished round score 64:37, got %d:%d", round.MyScore, round.OpponentScore)
	}
	if match.MyScore != 4 || match.OpponentScore != 2 {
		t.Fatalf("expected match frame score 4:2, got %d:%d", match.MyScore, match.OpponentScore)
	}
	if match.CurrentFrameMyScore != 0 || match.CurrentFrameOpponentScore != 0 {
		t.Fatalf("expected current frame reset to 0:0, got %d:%d", match.CurrentFrameMyScore, match.CurrentFrameOpponentScore)
	}
	if match.CurrentFrameStarted {
		t.Fatalf("expected current frame to be closed after finalize")
	}
}

func TestReopenSnookerFrameRestoresCurrentFrameAfterUndo(t *testing.T) {
	match := &model.Match{
		GameType:                  1,
		MyScore:                   4,
		OpponentScore:             2,
		CurrentFrameMyScore:       0,
		CurrentFrameOpponentScore: 0,
		CurrentFrameStarted:       false,
	}
	round := &model.MatchRound{
		MyScore:       72,
		OpponentScore: 35,
	}

	reopenSnookerFrame(match, round, 1)

	if match.MyScore != 3 || match.OpponentScore != 2 {
		t.Fatalf("expected match frame score rolled back to 3:2, got %d:%d", match.MyScore, match.OpponentScore)
	}
	if match.CurrentFrameMyScore != 72 || match.CurrentFrameOpponentScore != 35 {
		t.Fatalf("expected restored current frame score 72:35, got %d:%d", match.CurrentFrameMyScore, match.CurrentFrameOpponentScore)
	}
	if !match.CurrentFrameStarted {
		t.Fatalf("expected current frame reopened after undo")
	}
}
