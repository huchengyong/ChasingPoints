package match

import (
	"context"
	"reflect"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newMatchRefereeTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.Match{}, &model.MatchRound{}, &model.MatchAction{}); err != nil {
		t.Fatalf("prepare match referee schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:         db,
		MatchModel: model.NewMatchModel(db),
	}
}

func matchRefereeCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func TestGetCurrentMatchAllowsRefereeToResumeBoundMatch(t *testing.T) {
	svcCtx := newMatchRefereeTestSvc(t)
	opponentID := int64(2002)
	refereeID := int64(3003)
	now := time.Date(2026, 4, 8, 21, 0, 0, 0, time.UTC)

	if err := svcCtx.MatchModel.Create(&model.Match{
		Id:             81,
		UserId:         1001,
		OpponentId:     &opponentID,
		OpponentName:   "对手甲",
		GameType:       3,
		Status:         1,
		MyScore:        4,
		OpponentScore:  2,
		SyncRevision:   6,
		MatchTime:      now,
		RefereeUserId:  &refereeID,
		RefereeJoinedAt: &now,
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}

	resp, err := NewGetCurrentMatchLogic(matchRefereeCtx(refereeID), svcCtx).GetCurrentMatch()
	if err != nil {
		t.Fatalf("get current match: %v", err)
	}
	if !resp.Success || resp.Match == nil {
		t.Fatalf("expected referee to get current match, got %#v", resp)
	}
	assertStructFieldEqual(t, resp.Match, "ViewerRole", "referee")
	assertStructFieldEqual(t, resp.Match, "CanScore", true)
}

func TestMatchScoreRejectsPlayerWritesAfterRefereeBinding(t *testing.T) {
	svcCtx := newMatchRefereeTestSvc(t)
	opponentID := int64(2002)
	refereeID := int64(3003)
	now := time.Date(2026, 4, 8, 21, 5, 0, 0, time.UTC)

	if err := svcCtx.MatchModel.Create(&model.Match{
		Id:                         82,
		UserId:                     1001,
		OpponentId:                 &opponentID,
		OpponentName:               "对手乙",
		GameType:                   3,
		Status:                     1,
		MyScore:                    4,
		OpponentScore:              2,
		CurrentFrameStarted:        true,
		CurrentFrameMyScore:        0,
		CurrentFrameOpponentScore:  0,
		SyncRevision:               6,
		MatchTime:                  now,
		RefereeUserId:              &refereeID,
		RefereeJoinedAt:            &now,
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}

	resp, err := NewMatchScoreLogic(matchRefereeCtx(1001), svcCtx).MatchScore(&types.MatchScoreReq{
		MatchId:        82,
		Actor:          1,
		Score:          1,
		ClientActionId: "score-player-should-fail",
		BaseRevision:   6,
	})
	if err != nil {
		t.Fatalf("match score: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected player write to be rejected after referee binding, got %#v", resp)
	}
	if resp.Message != "本场已由裁判接管记分" {
		t.Fatalf("unexpected response message: %#v", resp)
	}
	if reflect.ValueOf(resp.Snapshot).IsZero() {
		t.Fatalf("expected rejection response to carry latest snapshot")
	}
	assertStructFieldEqual(t, &resp.Snapshot, "ViewerRole", "player1")
	assertStructFieldEqual(t, &resp.Snapshot, "CanScore", false)
}
