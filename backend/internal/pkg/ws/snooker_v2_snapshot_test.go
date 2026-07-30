package ws

import (
	"testing"

	"chasing_points/internal/model"
)

func TestSnookerV2SyncCarriesAuthoritativeStateForBothPlayers(t *testing.T) {
	opponentID := int64(202)
	match := &model.Match{
		Id:                        88,
		UserId:                    101,
		OpponentId:                &opponentID,
		GameType:                  1,
		SnookerRulesVersion:       model.SnookerRulesVersionWPBSA,
		BestOfFrames:              7,
		StartingActor:             2,
		MyScore:                   2,
		OpponentScore:             1,
		CurrentFrameMyScore:       18,
		CurrentFrameOpponentScore: 24,
		CurrentFrameStarted:       true,
		Status:                    1,
		SyncRevision:              9,
	}
	state := model.SnookerRoundState{
		RulesVersion:           model.SnookerRulesVersionWPBSA,
		Phase:                  model.SnookerPhaseColors,
		BallOn:                 model.SnookerBallBlue,
		Striker:                2,
		VisitNo:                6,
		CurrentBreak:           24,
		RedsRemaining:          0,
		FreeBallAvailable:      true,
		CueBallInHand:          true,
		MissWarningActive:      true,
		ClearedColors:          []int{2, 3, 4},
		RedBallCount:           15,
		ClearanceStarted:       true,
		ExpectedClearanceScore: 5,
	}

	player1 := buildMatchSyncDataForViewer(match, 101, 2, state, nil)
	player2 := buildMatchSyncDataForViewer(match, 202, 2, state, nil)
	for _, snapshot := range []MatchSyncData{player1, player2} {
		if snapshot.SnookerRulesVersion != model.SnookerRulesVersionWPBSA || snapshot.BestOfFrames != 7 ||
			snapshot.StartingActor != 2 || snapshot.SnookerPhase != model.SnookerPhaseColors || snapshot.SnookerBallOn != model.SnookerBallBlue ||
			snapshot.SnookerStriker != 2 || snapshot.SnookerVisitNo != 6 || snapshot.SnookerCurrentBreak != 24 ||
			snapshot.SnookerRedsRemaining != 0 || !snapshot.SnookerFreeBallAvailable || !snapshot.SnookerCueBallInHand || !snapshot.SnookerMissWarningActive {
			t.Fatalf("missing v2 state: %+v", snapshot)
		}
		if snapshot.CanFinish || snapshot.CanRequestFinish {
			t.Fatalf("v2 sync must disable generic finish: %+v", snapshot)
		}
	}
	if player1.MyScore != 2 || player1.CurrentFrameMyScore != 18 || player2.MyScore != 1 || player2.CurrentFrameMyScore != 24 {
		t.Fatalf("viewer scores are inconsistent: player1=%+v player2=%+v", player1, player2)
	}
}
