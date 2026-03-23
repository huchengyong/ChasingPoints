package logic

import (
	"testing"
	"time"

	"billiard_master/internal/model"
	"billiard_master/internal/types"
)

func TestBuildCurrentMatchInfoIncludesSnookerCurrentFrameFromViewerPerspective(t *testing.T) {
	now := time.Date(2026, 3, 17, 20, 0, 0, 0, time.FixedZone("CST", 8*3600))
	opponentID := int64(2002)
	match := &model.Match{
		Id:                        18,
		UserId:                    1001,
		OpponentId:                &opponentID,
		OpponentName:              "对手甲",
		GameType:                  1,
		GameMode:                  "",
		MyScore:                   3,
		OpponentScore:             2,
		CurrentFrameMyScore:       46,
		CurrentFrameOpponentScore: 33,
		CurrentFrameStarted:       true,
		SyncRevision:              7,
		MatchTime:                 now.Add(-18 * time.Minute),
	}

	info := buildCurrentMatchInfo(nil, 1001, match)
	assertCurrentMatchInfo(t, info, 3, 2, 46, 33, true)

	info = buildCurrentMatchInfo(nil, opponentID, match)
	assertCurrentMatchInfo(t, info, 2, 3, 33, 46, true)
}

func assertCurrentMatchInfo(t *testing.T, info *types.CurrentMatchInfo, myScore, opponentScore, frameMy, frameOpponent int, started bool) {
	t.Helper()
	if info == nil {
		t.Fatalf("expected current match info")
	}
	if info.MyScore != myScore || info.OpponentScore != opponentScore {
		t.Fatalf("expected match score %d:%d, got %d:%d", myScore, opponentScore, info.MyScore, info.OpponentScore)
	}
	if info.CurrentFrameMyScore != frameMy || info.CurrentFrameOpponentScore != frameOpponent {
		t.Fatalf("expected current frame score %d:%d, got %d:%d", frameMy, frameOpponent, info.CurrentFrameMyScore, info.CurrentFrameOpponentScore)
	}
	if info.CurrentFrameStarted != started {
		t.Fatalf("expected current_frame_started=%v, got %v", started, info.CurrentFrameStarted)
	}
	if info.ServerRevision != 7 {
		t.Fatalf("expected server_revision=7, got %d", info.ServerRevision)
	}
}
