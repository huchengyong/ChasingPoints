package logic

import (
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func TestBuildRoundRecordsCarriesWinType(t *testing.T) {
	winner := 2
	records := BuildRoundRecords([]model.MatchRound{
		{RoundNo: 1, MyScore: 8, OpponentScore: 3, Winner: &winner, WinType: "small_gold"},
	})
	if len(records) != 1 {
		t.Fatalf("unexpected records: %+v", records)
	}
	want := types.RoundRecord{RoundNumber: 1, Player1Score: 8, Player2Score: 3, Winner: 2, WinType: "small_gold"}
	if records[0] != want {
		t.Fatalf("round record must carry win_type: %+v", records[0])
	}
}
