package match

import (
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func TestScanStartOnlyClearsParticipantsWaitingAfterCreation(t *testing.T) {
	for _, busy := range []bool{false, true} {
		name := "waiting_player_is_scanned_opponent"
		if busy {
			name = "blocked_scan_keeps_waiting"
		}
		t.Run(name, func(t *testing.T) {
			svcCtx := newStartMatchChallengeTestSvc(t)
			for _, id := range []int64{1001, 2002, 3003, 4004} {
				if err := svcCtx.UserModel.Create(&model.User{Id: id, Nickname: "player"}); err != nil {
					t.Fatal(err)
				}
			}
			waiter := int64(1001)
			challenge := &model.Challenge{Id: 77, FromUserId: waiter, ToUserId: 2002, Status: model.ChallengeStatusAccepted, WaitingUserId: &waiter, ExpiresAt: time.Now().Add(time.Hour)}
			if err := svcCtx.ChallengeModel.Create(challenge); err != nil {
				t.Fatal(err)
			}
			caller, opponent := int64(3003), waiter
			if busy {
				other := int64(4004)
				if err := svcCtx.MatchModel.Create(&model.Match{Id: 81, UserId: 3003, OpponentId: &other, OpponentName: "D", GameType: 3, MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate, Status: 1, MatchTime: time.Now()}); err != nil {
					t.Fatal(err)
				}
				caller, opponent = waiter, 3003
			}
			resp, err := NewStartMatchLogic(startReputationCtx(caller), svcCtx).StartMatch(&types.StartMatchReq{GameType: 3, OpponentId: opponent, OpponentName: "opponent", MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate, MatchFormat: "free"})
			if err != nil || !resp.Success {
				t.Fatalf("start: %+v err=%v", resp, err)
			}
			stored, err := svcCtx.ChallengeModel.FindById(challenge.Id)
			if err != nil {
				t.Fatal(err)
			}
			if !busy && (resp.Action != startMatchActionCreated || stored.WaitingUserId != nil) {
				t.Fatalf("开局后对手的旧等待须清除: action=%s challenge=%+v", resp.Action, stored)
			}
			if busy && (resp.Action != startMatchActionBlocked || stored.WaitingUserId == nil) {
				t.Fatalf("开局受阻时旧等待须保留: action=%s challenge=%+v", resp.Action, stored)
			}
		})
	}
}

func TestPlayerFinishAttributionOverridesRefereeBinding(t *testing.T) {
	svcCtx := newStartMatchChallengeTestSvc(t)
	for _, id := range []int64{1001, 2002, 3003} {
		if err := svcCtx.UserModel.Create(&model.User{Id: id}); err != nil {
			t.Fatal(err)
		}
	}
	challengeID, opponent, referee := int64(77), int64(2002), int64(3003)
	if err := svcCtx.ChallengeModel.Create(&model.Challenge{Id: challengeID, FromUserId: 1001, ToUserId: opponent, Status: model.ChallengeStatusStarted, ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	match := &model.Match{Id: 89, UserId: 1001, OpponentId: &opponent, RefereeUserId: &referee, ChallengeId: &challengeID, OpponentName: "B", GameType: 3, MatchFormat: "free", MyScore: 1, MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate, Status: 1, MatchTime: time.Now()}
	if err := svcCtx.MatchModel.Create(match); err != nil {
		t.Fatal(err)
	}
	winner := 1
	if err := svcCtx.DB.Create(&model.MatchRound{MatchId: match.Id, RoundNo: 1, Winner: &winner, WinType: "normal", MyScore: 1}).Error; err != nil {
		t.Fatal(err)
	}
	resp, err := NewFinishMatchLogic(startReputationCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{MatchId: match.Id, ClientActionId: "player-attr", BaseRevision: 0})
	if err != nil || !resp.Success {
		t.Fatalf("player finish: %+v err=%v", resp, err)
	}
	saved, err := svcCtx.MatchModel.FindById(match.Id)
	if err != nil || saved.CompletedByUserId == nil || *saved.CompletedByUserId != 1001 || saved.CompletionSource != model.CompletionSourcePlayerDirect {
		t.Fatalf("参赛者结束必须归因参赛者而非裁判: completed_by=%v source=%s err=%v", saved.CompletedByUserId, saved.CompletionSource, err)
	}
}

func TestRefereeBoundChallengePlayerFinishReplay(t *testing.T) {
	svcCtx := newStartMatchChallengeTestSvc(t)
	for _, id := range []int64{1001, 2002, 3003} {
		if err := svcCtx.UserModel.Create(&model.User{Id: id, Nickname: "player"}); err != nil {
			t.Fatal(err)
		}
	}
	challengeID, opponent, referee := int64(77), int64(2002), int64(3003)
	if err := svcCtx.ChallengeModel.Create(&model.Challenge{Id: challengeID, FromUserId: 1001, ToUserId: opponent, Status: model.ChallengeStatusStarted, ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	match := &model.Match{Id: 88, UserId: 1001, OpponentId: &opponent, RefereeUserId: &referee, ChallengeId: &challengeID, OpponentName: "B", GameType: 3, MatchFormat: "free", MyScore: 1, MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate, Status: 1, MatchTime: time.Now()}
	if err := svcCtx.MatchModel.Create(match); err != nil {
		t.Fatal(err)
	}
	winner := 1
	if err := svcCtx.DB.Create(&model.MatchRound{MatchId: match.Id, RoundNo: 1, Winner: &winner, WinType: "normal", MyScore: 1}).Error; err != nil {
		t.Fatal(err)
	}
	req := &types.FinishMatchReq{MatchId: match.Id, ClientActionId: "player-direct", BaseRevision: 0}
	logic := NewFinishMatchLogic(startReputationCtx(1001), svcCtx)
	first, err := logic.FinishMatch(req)
	if err != nil || !first.Success {
		t.Fatalf("first finish: %+v err=%v", first, err)
	}
	again, err := logic.FinishMatch(req)
	if err != nil || !again.Success || !again.Accepted {
		t.Fatalf("同一动作重试须幂等成功: %+v err=%v", again, err)
	}
}
