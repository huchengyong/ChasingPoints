package match

import (
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

const (
	timeoutRequesterID = int64(1001)
	timeoutOpponentID  = int64(2002)
)

func newFinishTimeoutTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()
	svcCtx := newFinishMatchReputationTestSvc(t)
	if err := svcCtx.DB.AutoMigrate(&model.Challenge{}); err != nil {
		t.Fatalf("prepare challenge schema: %v", err)
	}
	svcCtx.ChallengeModel = model.NewChallengeModel(svcCtx.DB)
	seedFinishReputationUsers(t, svcCtx,
		model.User{Id: timeoutRequesterID, Nickname: "小李"},
		model.User{Id: timeoutOpponentID, Nickname: "小王"},
	)
	return svcCtx
}

// seedPendingRankedFinish 创建一场5:3的排位比赛，结束请求由 timeoutRequesterID 在 age 之前发起。
func seedPendingRankedFinish(t *testing.T, svcCtx *svc.ServiceContext, matchID int64, age time.Duration) {
	t.Helper()
	opponentID := timeoutOpponentID
	requesterID := timeoutRequesterID
	requestedAt := time.Now().Add(-age)
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id: matchID, UserId: timeoutRequesterID, OpponentId: &opponentID, OpponentName: "小王", GameType: 3,
		MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic, FinishConfirmationRequired: true,
		FinishState: model.FinishStatePendingConfirmation, FinishRequestedBy: &requesterID, FinishRequestedAt: &requestedAt,
		FinishRequestRevision: 3, SyncRevision: 3, Status: 1, MyScore: 5, OpponentScore: 3,
		CurrentFrameStarted: true, MatchTime: time.Now().Add(-time.Hour),
	}); err != nil {
		t.Fatalf("create pending match: %v", err)
	}
}

func countMatchActions(t *testing.T, svcCtx *svc.ServiceContext, matchID int64, actionType string) int64 {
	t.Helper()
	var count int64
	if err := svcCtx.DB.Model(&model.MatchAction{}).Where("match_id = ? AND action_type = ?", matchID, actionType).Count(&count).Error; err != nil {
		t.Fatalf("count %s actions: %v", actionType, err)
	}
	return count
}

func TestFinishActionsAfterDeadlineCompleteByTimeoutAndAreRejected(t *testing.T) {
	cases := []struct {
		name   string
		action string
		run    func(svcCtx *svc.ServiceContext, req *types.FinishMatchActionReq) (*types.FinishMatchActionResp, error)
	}{
		{"opponent confirms late", "finish_confirm", func(svcCtx *svc.ServiceContext, req *types.FinishMatchActionReq) (*types.FinishMatchActionResp, error) {
			return NewConfirmFinishMatchLogic(finishReputationCtx(timeoutOpponentID), svcCtx).ConfirmFinishMatch(req)
		}},
		{"opponent disputes late", "finish_dispute", func(svcCtx *svc.ServiceContext, req *types.FinishMatchActionReq) (*types.FinishMatchActionResp, error) {
			return NewDisputeFinishMatchLogic(finishReputationCtx(timeoutOpponentID), svcCtx).DisputeFinishMatch(req)
		}},
		{"requester withdraws late", "finish_withdraw", func(svcCtx *svc.ServiceContext, req *types.FinishMatchActionReq) (*types.FinishMatchActionResp, error) {
			return NewWithdrawFinishMatchLogic(finishReputationCtx(timeoutRequesterID), svcCtx).WithdrawFinishMatch(req)
		}},
	}
	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svcCtx := newFinishTimeoutTestSvc(t)
			matchID := int64(9500 + index)
			seedPendingRankedFinish(t, svcCtx, matchID, model.FinishRequestTTL+time.Second)

			resp, err := tc.run(svcCtx, &types.FinishMatchActionReq{MatchId: matchID, ClientActionId: "late-action", BaseRevision: 3})
			if err != nil || resp.Success || resp.Message != errFinishRequestTimedOut.Error() {
				t.Fatalf("late action must be rejected with the timeout message: resp=%#v err=%v", resp, err)
			}
			if resp.Snapshot.Status != 2 {
				t.Fatalf("rejection must carry the completed authoritative snapshot: %+v", resp.Snapshot)
			}
			stored, err := svcCtx.MatchModel.FindById(matchID)
			if err != nil || stored.Status != 2 || stored.Result == nil || *stored.Result != 1 {
				t.Fatalf("late action must first complete the match with the frozen 5:3: %+v err=%v", stored, err)
			}
			if stored.CompletionSource != model.CompletionSourceFinishTimeout || stored.CompletedByUserId == nil || *stored.CompletedByUserId != timeoutRequesterID {
				t.Fatalf("completion = %q by %v, want finish_timeout by requester", stored.CompletionSource, stored.CompletedByUserId)
			}
			if got := countMatchActions(t, svcCtx, matchID, finishTimeoutActionType); got != 1 {
				t.Fatalf("finish_timeout actions = %d, want 1", got)
			}
			if got := countMatchActions(t, svcCtx, matchID, tc.action); got != 0 {
				t.Fatalf("late %s must not be recorded, got %d", tc.action, got)
			}
		})
	}
}

func TestConfirmOneSecondBeforeDeadlineStillConfirms(t *testing.T) {
	svcCtx := newFinishTimeoutTestSvc(t)
	seedPendingRankedFinish(t, svcCtx, 9510, model.FinishRequestTTL-time.Second)

	resp, err := NewConfirmFinishMatchLogic(finishReputationCtx(timeoutOpponentID), svcCtx).ConfirmFinishMatch(&types.FinishMatchActionReq{
		MatchId: 9510, ClientActionId: "confirm-in-time", BaseRevision: 3,
	})
	if err != nil || !resp.Success {
		t.Fatalf("confirm at 29s must succeed: resp=%#v err=%v", resp, err)
	}
	stored, err := svcCtx.MatchModel.FindById(9510)
	if err != nil || stored.Status != 2 || stored.CompletionSource != model.CompletionSourcePlayerConfirmed {
		t.Fatalf("confirmed match = %+v err=%v", stored, err)
	}
	// 确认路径的完成人是确认方；发起方即另一名参赛者。
	if stored.CompletedByUserId == nil || *stored.CompletedByUserId != timeoutOpponentID {
		t.Fatalf("confirmed match must record the confirmer, got %v", stored.CompletedByUserId)
	}
	if got := countMatchActions(t, svcCtx, 9510, finishTimeoutActionType); got != 0 {
		t.Fatalf("in-time confirm must not write finish_timeout, got %d", got)
	}
	// 之后的到期扫描不会再结算一次。
	if completion, err := completeExpiredFinishRequest(finishReputationCtx(0), svcCtx, 9510, time.Now().Add(time.Minute)); err != nil || completion != nil {
		t.Fatalf("completed match must be skipped by expiry: completion=%+v err=%v", completion, err)
	}
}

func TestChallengeRankedFinishStartsCountdownInsteadOfFinishing(t *testing.T) {
	svcCtx := newFinishTimeoutTestSvc(t)
	challengeID := int64(77)
	if err := svcCtx.ChallengeModel.Create(&model.Challenge{
		Id: challengeID, FromUserId: timeoutRequesterID, ToUserId: timeoutOpponentID,
		Status: model.ChallengeStatusStarted, ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("seed challenge: %v", err)
	}
	opponentID := timeoutOpponentID
	seedFinishReputationMatch(t, svcCtx, model.Match{
		Id: 9520, UserId: timeoutRequesterID, OpponentId: &opponentID, OpponentName: "小王", GameType: 3,
		MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic, FinishConfirmationRequired: true,
		FinishState: model.FinishStateNone, Status: 1, MyScore: 5, OpponentScore: 3, ChallengeId: &challengeID,
		CurrentFrameStarted: true, MatchTime: time.Now().Add(-time.Hour),
	}, 1)

	direct, err := NewFinishMatchLogic(finishReputationCtx(timeoutRequesterID), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId: 9520, ClientActionId: "direct-finish",
	})
	if err != nil || direct.Success {
		t.Fatalf("challenge ranked match must not finish directly: resp=%#v err=%v", direct, err)
	}
	resp, err := NewRequestFinishMatchLogic(finishReputationCtx(timeoutRequesterID), svcCtx).RequestFinishMatch(&types.FinishMatchActionReq{
		MatchId: 9520, ClientActionId: "request-finish", BaseRevision: direct.ServerRevision,
	})
	if err != nil || !resp.Success {
		t.Fatalf("challenge ranked request must succeed: resp=%#v err=%v", resp, err)
	}
	if resp.Snapshot.FinishState != model.FinishStatePendingConfirmation || resp.Snapshot.FinishRequestRemainingSeconds != int(model.FinishRequestTTL.Seconds()) ||
		resp.Snapshot.FinishRequestExpiresAt == "" || !resp.Snapshot.CanWithdrawFinish || resp.Snapshot.CanConfirmFinish {
		t.Fatalf("requester snapshot must show the full countdown and withdraw only: %+v", resp.Snapshot)
	}
	challenge, err := svcCtx.ChallengeModel.FindById(challengeID)
	if err != nil || challenge.Status != model.ChallengeStatusStarted {
		t.Fatalf("challenge must stay started during confirmation: %+v err=%v", challenge, err)
	}
}

func TestPracticeChallengeDirectFinishNotifiesOtherPlayerOnce(t *testing.T) {
	svcCtx := newFinishTimeoutTestSvc(t)
	challengeID := int64(78)
	if err := svcCtx.ChallengeModel.Create(&model.Challenge{
		Id: challengeID, FromUserId: timeoutRequesterID, ToUserId: timeoutOpponentID,
		Status: model.ChallengeStatusStarted, ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("seed challenge: %v", err)
	}
	opponentID := timeoutOpponentID
	seedFinishReputationMatch(t, svcCtx, model.Match{
		Id: 9530, UserId: timeoutRequesterID, OpponentId: &opponentID, OpponentName: "小王", GameType: 3,
		MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate, FinishState: model.FinishStateNone,
		Status: 1, MyScore: 3, OpponentScore: 1, ChallengeId: &challengeID, CurrentFrameStarted: true, MatchTime: time.Now().Add(-time.Hour),
	}, 1)

	resp, err := NewFinishMatchLogic(finishReputationCtx(timeoutRequesterID), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId: 9530, ClientActionId: "practice-finish",
	})
	if err != nil || !resp.Success {
		t.Fatalf("practice challenge must finish directly: resp=%#v err=%v", resp, err)
	}
	var notices []model.Notification
	if err := svcCtx.DB.Where("user_id = ? AND type = ?", timeoutOpponentID, "match_result").Find(&notices).Error; err != nil {
		t.Fatalf("load notices: %v", err)
	}
	if len(notices) != 1 || notices[0].Content != "小李结束了对局 · 1:3" {
		t.Fatalf("other player must get exactly one finisher notice, got %#v", notices)
	}
	challenge, err := svcCtx.ChallengeModel.FindById(challengeID)
	if err != nil || challenge.Status != model.ChallengeStatusCompleted {
		t.Fatalf("practice finish must complete the challenge: %+v err=%v", challenge, err)
	}
}
