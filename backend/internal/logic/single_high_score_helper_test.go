package logic

import (
	"testing"
	"time"

	"chasing_points/internal/model"
)

func TestBuildSingleHighScoreRecordsUsesSnookerBreakScoresAndViewerRole(t *testing.T) {
	records := buildSingleHighScoreRecords([]singleHighScoreCandidate{
		{
			MatchId:      11,
			GameType:     1,
			OpponentName: "对手甲",
			Date:         time.Date(2026, 3, 16, 20, 0, 0, 0, time.UTC),
			ViewerActor:  1,
			Actions: []model.MatchAction{
				{RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 1},
				{RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 7},
				{RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 1},
				{RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 7},
				{RoundNo: 1, ActionType: "foul", Actor: 1, ScoreChange: 4},
			},
		},
		{
			MatchId:      12,
			GameType:     1,
			OpponentName: "对手乙",
			Date:         time.Date(2026, 3, 17, 21, 0, 0, 0, time.UTC),
			ViewerActor:  2,
			Actions: []model.MatchAction{
				{RoundNo: 1, ActionType: "score", Actor: 2, ScoreChange: 1},
				{RoundNo: 1, ActionType: "score", Actor: 2, ScoreChange: 7},
				{RoundNo: 1, ActionType: "score", Actor: 2, ScoreChange: 1},
				{RoundNo: 1, ActionType: "score", Actor: 2, ScoreChange: 6},
			},
		},
	}, 10)

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].MatchId != 11 || records[0].Score != 16 {
		t.Fatalf("expected top record match 11 score 16, got %+v", records[0])
	}
	if records[1].MatchId != 12 || records[1].Score != 15 {
		t.Fatalf("expected second record match 12 score 15, got %+v", records[1])
	}
}
