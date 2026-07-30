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
	if err := db.AutoMigrate(&model.User{}, &model.Match{}, &model.MatchRound{}, &model.MatchAction{}, &model.MatchAchievement{}); err != nil {
		t.Fatalf("prepare match referee schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:         db,
		UserModel:  model.NewUserModel(db),
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
	seedCurrentMatchInfoUser(t, svcCtx, 1001, "选手甲", "player1.png")
	seedCurrentMatchInfoUser(t, svcCtx, opponentID, "选手乙", "player2.png")
	seedCurrentMatchInfoUser(t, svcCtx, refereeID, "裁判丙", "referee.png")

	if err := svcCtx.MatchModel.Create(&model.Match{
		Id:              81,
		UserId:          1001,
		OpponentId:      &opponentID,
		OpponentName:    "对手甲",
		GameType:        3,
		Status:          1,
		MyScore:         4,
		OpponentScore:   2,
		SyncRevision:    6,
		MatchTime:       now,
		RefereeUserId:   &refereeID,
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
	assertFixedParticipants(t, resp.Match, 1001, "选手甲", "player1.png", opponentID, "选手乙", "player2.png")
}

func TestMatchScoreRejectsPlayerWritesAfterRefereeBinding(t *testing.T) {
	svcCtx := newMatchRefereeTestSvc(t)
	opponentID := int64(2002)
	refereeID := int64(3003)
	now := time.Date(2026, 4, 8, 21, 5, 0, 0, time.UTC)

	if err := svcCtx.MatchModel.Create(&model.Match{
		Id:                        82,
		UserId:                    1001,
		OpponentId:                &opponentID,
		OpponentName:              "对手乙",
		GameType:                  3,
		Status:                    1,
		MyScore:                   4,
		OpponentScore:             2,
		CurrentFrameStarted:       true,
		CurrentFrameMyScore:       0,
		CurrentFrameOpponentScore: 0,
		SyncRevision:              6,
		MatchTime:                 now,
		RefereeUserId:             &refereeID,
		RefereeJoinedAt:           &now,
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

func TestRefereeCanWriteBothFixedParticipants(t *testing.T) {
	svcCtx := newMatchRefereeTestSvc(t)
	opponentID := int64(2002)
	refereeID := int64(3003)
	now := time.Date(2026, 4, 8, 21, 10, 0, 0, time.UTC)

	if err := svcCtx.MatchModel.Create(&model.Match{
		Id:                  83,
		UserId:              1001,
		OpponentId:          &opponentID,
		OpponentName:        "选手乙",
		GameType:            2,
		Status:              1,
		CurrentFrameStarted: true,
		MatchTime:           now,
		RefereeUserId:       &refereeID,
		RefereeJoinedAt:     &now,
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}

	requests := []struct {
		name        string
		call        func() (bool, error)
		wantPlayer1 int
		wantPlayer2 int
	}{
		{
			name: "score player1",
			call: func() (bool, error) {
				resp, err := NewMatchScoreLogic(matchRefereeCtx(refereeID), svcCtx).MatchScore(&types.MatchScoreReq{MatchId: 83, Actor: 1, Score: 2, ClientActionId: "referee-score-player1", BaseRevision: 0})
				return resp.Success, err
			},
			wantPlayer1: 2,
			wantPlayer2: 0,
		},
		{
			name: "score player2",
			call: func() (bool, error) {
				resp, err := NewMatchScoreLogic(matchRefereeCtx(refereeID), svcCtx).MatchScore(&types.MatchScoreReq{MatchId: 83, Actor: 2, Score: 3, ClientActionId: "referee-score-player2", BaseRevision: 1})
				return resp.Success, err
			},
			wantPlayer1: 2,
			wantPlayer2: 3,
		},
		{
			name: "player1 foul scores player2",
			call: func() (bool, error) {
				resp, err := NewMatchFoulLogic(matchRefereeCtx(refereeID), svcCtx).MatchFoul(&types.MatchFoulReq{MatchId: 83, Actor: 1, Score: 1, ClientActionId: "referee-foul-player1", BaseRevision: 2})
				return resp.Success, err
			},
			wantPlayer1: 2,
			wantPlayer2: 4,
		},
		{
			name: "player2 foul scores player1",
			call: func() (bool, error) {
				resp, err := NewMatchFoulLogic(matchRefereeCtx(refereeID), svcCtx).MatchFoul(&types.MatchFoulReq{MatchId: 83, Actor: 2, Score: 1, ClientActionId: "referee-foul-player2", BaseRevision: 3})
				return resp.Success, err
			},
			wantPlayer1: 3,
			wantPlayer2: 4,
		},
		{
			name: "player1 wins round",
			call: func() (bool, error) {
				resp, err := NewEndRoundLogic(matchRefereeCtx(refereeID), svcCtx).EndRound(&types.EndRoundReq{MatchId: 83, Winner: 1, WinType: "normal", Score: 4, ClientActionId: "referee-win-player1", BaseRevision: 4})
				return resp.Success, err
			},
			wantPlayer1: 7,
			wantPlayer2: 4,
		},
		{
			name: "player2 wins round",
			call: func() (bool, error) {
				resp, err := NewEndRoundLogic(matchRefereeCtx(refereeID), svcCtx).EndRound(&types.EndRoundReq{MatchId: 83, Winner: 2, WinType: "small_gold", Score: 7, ClientActionId: "referee-win-player2", BaseRevision: 5})
				return resp.Success, err
			},
			wantPlayer1: 7,
			wantPlayer2: 11,
		},
	}

	for _, request := range requests {
		success, err := request.call()
		if err != nil {
			t.Fatalf("%s: %v", request.name, err)
		}
		if !success {
			t.Fatalf("%s: expected success", request.name)
		}
		match, err := svcCtx.MatchModel.FindById(83)
		if err != nil {
			t.Fatalf("%s: reload match: %v", request.name, err)
		}
		if match.MyScore != request.wantPlayer1 || match.OpponentScore != request.wantPlayer2 {
			t.Fatalf("%s: got player1=%d player2=%d, want player1=%d player2=%d", request.name, match.MyScore, match.OpponentScore, request.wantPlayer1, request.wantPlayer2)
		}
	}
}
