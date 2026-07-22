package match

import (
	"encoding/json"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg/ws"
	"chasing_points/internal/types"
)

func TestRankedFinishRequestConfirmCompletesOnlyAfterOpponentAction(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	opponentID := int64(2002)
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id:                         9001,
		UserId:                     1001,
		OpponentId:                 &opponentID,
		OpponentName:               "对手",
		GameType:                   3,
		MatchMode:                  model.MatchModeRanked,
		FinishConfirmationRequired: true,
		Visibility:                 model.MatchVisibilityPublic,
		FinishState:                model.FinishStateNone,
		Status:                     1,
		MyScore:                    2,
		OpponentScore:              2,
		CurrentFrameStarted:        true,
		MatchTime:                  time.Now(),
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}

	request, err := NewRequestFinishMatchLogic(finishReputationCtx(1001), svcCtx).RequestFinishMatch(&types.FinishMatchActionReq{
		MatchId: 9001, ClientActionId: "finish-request-9001", BaseRevision: 0,
	})
	if err != nil || !request.Success || request.FinishState != model.FinishStatePendingConfirmation {
		t.Fatalf("unexpected request response: resp=%#v err=%v", request, err)
	}

	stored, err := svcCtx.MatchModel.FindById(9001)
	if err != nil || stored == nil || stored.Status != 1 || stored.FinishState != model.FinishStatePendingConfirmation {
		t.Fatalf("expected pending match, stored=%+v err=%v", stored, err)
	}

	confirm, err := NewConfirmFinishMatchLogic(finishReputationCtx(opponentID), svcCtx).ConfirmFinishMatch(&types.FinishMatchActionReq{
		MatchId: 9001, ClientActionId: "finish-confirm-9001", BaseRevision: request.ServerRevision,
	})
	if err != nil || !confirm.Success || confirm.FinishState != model.FinishStateNone {
		t.Fatalf("unexpected confirm response: resp=%#v err=%v", confirm, err)
	}
	stored, err = svcCtx.MatchModel.FindById(9001)
	if err != nil || stored == nil || stored.Status != 2 || stored.FinishState != model.FinishStateNone {
		t.Fatalf("expected completed match after confirm, stored=%+v err=%v", stored, err)
	}
}

func TestFinishConfirmationFlagControlsRankedFinishFlow(t *testing.T) {
	opponentID := int64(2002)
	match := &model.Match{Id: 9013, UserId: 1001, OpponentId: &opponentID, Status: 1, MatchMode: model.MatchModeRanked}
	if shouldRequestRankedFinish(match, 1001) {
		t.Fatal("legacy ranked match should keep immediate finish compatibility")
	}
	match.FinishConfirmationRequired = true
	if !shouldRequestRankedFinish(match, 1001) {
		t.Fatal("ranked match with explicit confirmation flag should require confirmation")
	}
}

func TestRankedFinishActionsAreIdempotent(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	opponentID := int64(2002)
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id: 9004, UserId: 1001, OpponentId: &opponentID, OpponentName: "对手", GameType: 3,
		MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic, FinishConfirmationRequired: true, FinishState: model.FinishStateNone,
		Status: 1, MyScore: 3, OpponentScore: 1, CurrentFrameStarted: true, MatchTime: time.Now(),
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}

	requestLogic := NewRequestFinishMatchLogic(finishReputationCtx(1001), svcCtx)
	firstRequest, err := requestLogic.RequestFinishMatch(&types.FinishMatchActionReq{
		MatchId: 9004, ClientActionId: "finish-request-9004", BaseRevision: 0,
	})
	if err != nil || !firstRequest.Success {
		t.Fatalf("first request: resp=%#v err=%v", firstRequest, err)
	}
	secondRequest, err := requestLogic.RequestFinishMatch(&types.FinishMatchActionReq{
		MatchId: 9004, ClientActionId: "finish-request-9004", BaseRevision: 0,
	})
	if err != nil || !secondRequest.Success || secondRequest.ServerRevision != firstRequest.ServerRevision {
		t.Fatalf("duplicate request should replay original revision: first=%#v second=%#v err=%v", firstRequest, secondRequest, err)
	}

	confirmLogic := NewConfirmFinishMatchLogic(finishReputationCtx(opponentID), svcCtx)
	firstConfirm, err := confirmLogic.ConfirmFinishMatch(&types.FinishMatchActionReq{
		MatchId: 9004, ClientActionId: "finish-confirm-9004", BaseRevision: firstRequest.ServerRevision,
	})
	if err != nil || !firstConfirm.Success || firstConfirm.ServerRevision <= firstRequest.ServerRevision {
		t.Fatalf("first confirm: resp=%#v err=%v", firstConfirm, err)
	}
	secondConfirm, err := confirmLogic.ConfirmFinishMatch(&types.FinishMatchActionReq{
		MatchId: 9004, ClientActionId: "finish-confirm-9004", BaseRevision: firstRequest.ServerRevision,
	})
	if err != nil || !secondConfirm.Success || secondConfirm.ServerRevision != firstConfirm.ServerRevision {
		t.Fatalf("duplicate confirm should replay completed result: first=%#v second=%#v err=%v", firstConfirm, secondConfirm, err)
	}

	var endActionCount int64
	if err := svcCtx.DB.Model(&model.MatchAction{}).Where("match_id = ? AND action_type = ?", 9004, "match_end").Count(&endActionCount).Error; err != nil {
		t.Fatalf("count finish actions: %v", err)
	}
	if endActionCount != 1 {
		t.Fatalf("expected one settlement action after duplicate confirm, got %d", endActionCount)
	}
	var rankLogCount int64
	if err := svcCtx.DB.Model(&model.RankChangeLog{}).Where("match_id = ?", 9004).Count(&rankLogCount).Error; err != nil {
		t.Fatalf("count rank logs: %v", err)
	}
	if rankLogCount != 2 {
		t.Fatalf("expected one rank log per player, got %d", rankLogCount)
	}
}

func TestFinishActionClientIdCannotBeReusedForAnotherAction(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	opponentID := int64(2002)
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id: 9008, UserId: 1001, OpponentId: &opponentID, OpponentName: "对手", GameType: 3,
		MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic, FinishConfirmationRequired: true, FinishState: model.FinishStateNone,
		Status: 1, MyScore: 2, OpponentScore: 1, CurrentFrameStarted: true, MatchTime: time.Now(),
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}

	request, err := NewRequestFinishMatchLogic(finishReputationCtx(1001), svcCtx).RequestFinishMatch(&types.FinishMatchActionReq{
		MatchId: 9008, ClientActionId: "finish-reused-client-id", BaseRevision: 0,
	})
	if err != nil || !request.Success {
		t.Fatalf("request finish: resp=%#v err=%v", request, err)
	}
	confirm, err := NewConfirmFinishMatchLogic(finishReputationCtx(opponentID), svcCtx).ConfirmFinishMatch(&types.FinishMatchActionReq{
		MatchId: 9008, ClientActionId: "finish-reused-client-id", BaseRevision: request.ServerRevision,
	})
	if err != nil || confirm.Success {
		t.Fatalf("same client action id must not confirm a request action: resp=%#v err=%v", confirm, err)
	}

	stored, err := svcCtx.MatchModel.FindById(9008)
	if err != nil || stored == nil || stored.FinishState != model.FinishStatePendingConfirmation || stored.Status != 1 {
		t.Fatalf("reused client action id changed finish state: stored=%+v err=%v", stored, err)
	}
}

func TestConfirmSettlementFailureKeepsPendingFinishRequest(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	opponentID := int64(2002)
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id: 9009, UserId: 1001, OpponentId: &opponentID, OpponentName: "对手", GameType: 3,
		MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic, FinishConfirmationRequired: true, FinishState: model.FinishStateNone,
		Status: 1, MyScore: 2, OpponentScore: 1, CurrentFrameStarted: true, MatchTime: time.Now(),
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}
	request, err := NewRequestFinishMatchLogic(finishReputationCtx(1001), svcCtx).RequestFinishMatch(&types.FinishMatchActionReq{
		MatchId: 9009, ClientActionId: "finish-request-atomic-9009", BaseRevision: 0,
	})
	if err != nil || !request.Success {
		t.Fatalf("request finish: resp=%#v err=%v", request, err)
	}
	if err := svcCtx.DB.Exec("DROP TABLE rank_change_logs").Error; err != nil {
		t.Fatalf("drop rank logs to force settlement failure: %v", err)
	}

	confirm, err := NewConfirmFinishMatchLogic(finishReputationCtx(opponentID), svcCtx).ConfirmFinishMatch(&types.FinishMatchActionReq{
		MatchId: 9009, ClientActionId: "finish-confirm-atomic-9009", BaseRevision: request.ServerRevision,
	})
	if err != nil || confirm.Success {
		t.Fatalf("confirm should fail when settlement transaction fails: resp=%#v err=%v", confirm, err)
	}
	stored, err := svcCtx.MatchModel.FindById(9009)
	if err != nil || stored == nil || stored.Status != 1 || stored.FinishState != model.FinishStatePendingConfirmation {
		t.Fatalf("failed confirmation must leave pending request intact: stored=%+v err=%v", stored, err)
	}
	var confirmActionCount int64
	if err := svcCtx.DB.Model(&model.MatchAction{}).Where("match_id = ? AND action_type = ?", 9009, "finish_confirm").Count(&confirmActionCount).Error; err != nil {
		t.Fatalf("count confirm actions: %v", err)
	}
	if confirmActionCount != 0 {
		t.Fatalf("failed confirmation must not persist confirm action, got %d", confirmActionCount)
	}
}

func TestRankedFinishRequestCanBeDisputedAndWithdrawn(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	opponentID := int64(2002)
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id: 9002, UserId: 1001, OpponentId: &opponentID, OpponentName: "对手", GameType: 3,
		MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic, FinishConfirmationRequired: true, FinishState: model.FinishStateNone,
		Status: 1, CurrentFrameStarted: true, MatchTime: time.Now(),
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}
	request, _ := NewRequestFinishMatchLogic(finishReputationCtx(1001), svcCtx).RequestFinishMatch(&types.FinishMatchActionReq{MatchId: 9002, ClientActionId: "finish-request-9002", BaseRevision: 0})
	dispute, err := NewDisputeFinishMatchLogic(finishReputationCtx(opponentID), svcCtx).DisputeFinishMatch(&types.FinishMatchActionReq{MatchId: 9002, ClientActionId: "finish-dispute-9002", BaseRevision: request.ServerRevision})
	if err != nil || !dispute.Success || dispute.FinishState != model.FinishStateNone {
		t.Fatalf("unexpected dispute response: resp=%#v err=%v", dispute, err)
	}
	request, _ = NewRequestFinishMatchLogic(finishReputationCtx(1001), svcCtx).RequestFinishMatch(&types.FinishMatchActionReq{MatchId: 9002, ClientActionId: "finish-request-9002b", BaseRevision: dispute.ServerRevision})
	withdraw, err := NewWithdrawFinishMatchLogic(finishReputationCtx(1001), svcCtx).WithdrawFinishMatch(&types.FinishMatchActionReq{MatchId: 9002, ClientActionId: "finish-withdraw-9002", BaseRevision: request.ServerRevision})
	if err != nil || !withdraw.Success || withdraw.FinishState != model.FinishStateNone {
		t.Fatalf("unexpected withdraw response: resp=%#v err=%v", withdraw, err)
	}
}

func TestPracticeFinishCompletesWithoutCompetitiveSettlement(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	opponentID := int64(2002)
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id: 9003, UserId: 1001, OpponentId: &opponentID, OpponentName: "对手", GameType: 3,
		MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate, FinishState: model.FinishStateNone,
		Status: 1, MyScore: 3, OpponentScore: 1, CurrentFrameStarted: true, MatchTime: time.Now(),
	}); err != nil {
		t.Fatalf("create practice match: %v", err)
	}
	resp, err := NewFinishMatchLogic(finishReputationCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId: 9003, ClientActionId: "finish-practice-9003", BaseRevision: 0,
	})
	if err != nil || !resp.Success {
		t.Fatalf("finish practice: resp=%#v err=%v", resp, err)
	}
	stored, err := svcCtx.MatchModel.FindById(9003)
	if err != nil || stored == nil || stored.Status != 2 {
		t.Fatalf("expected practice match to finish, stored=%+v err=%v", stored, err)
	}
	var rankLogCount int64
	if err := svcCtx.DB.Model(&model.RankChangeLog{}).Where("match_id = ?", 9003).Count(&rankLogCount).Error; err != nil {
		t.Fatalf("count rank logs: %v", err)
	}
	if rankLogCount != 0 {
		t.Fatalf("practice match should not write rank logs, got %d", rankLogCount)
	}
}

func TestPracticeFinishDoesNotGrantMemberGrowth(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	if err := svcCtx.DB.AutoMigrate(&model.MemberGrowthProfile{}, &model.MemberGrowthLog{}); err != nil {
		t.Fatalf("prepare member growth schema: %v", err)
	}
	svcCtx.MemberGrowthProfileModel = model.NewMemberGrowthProfileModel(svcCtx.DB)
	svcCtx.MemberGrowthLogModel = model.NewMemberGrowthLogModel(svcCtx.DB)
	now := time.Now()
	memberExpiry := now.Add(24 * time.Hour)
	seedFinishReputationUsers(t, svcCtx,
		model.User{Id: 1001, Nickname: "选手甲", MemberExpiresAt: &memberExpiry},
		model.User{Id: 2002, Nickname: "选手乙", MemberExpiresAt: &memberExpiry},
	)
	opponentID := int64(2002)
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id: 9007, UserId: 1001, OpponentId: &opponentID, OpponentName: "选手乙", GameType: 3,
		MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate, FinishState: model.FinishStateNone,
		Status: 1, MyScore: 2, OpponentScore: 1, CurrentFrameStarted: true, MatchTime: now,
	}); err != nil {
		t.Fatalf("create practice match: %v", err)
	}

	resp, err := NewFinishMatchLogic(finishReputationCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId: 9007, ClientActionId: "finish-practice-9007", BaseRevision: 0,
	})
	if err != nil || !resp.Success {
		t.Fatalf("finish practice: resp=%#v err=%v", resp, err)
	}
	var growthLogCount int64
	if err := svcCtx.DB.Model(&model.MemberGrowthLog{}).Where("match_id = ?", 9007).Count(&growthLogCount).Error; err != nil {
		t.Fatalf("count growth logs: %v", err)
	}
	if growthLogCount != 0 {
		t.Fatalf("practice match should not grant member growth, got %d logs", growthLogCount)
	}
}

func TestRefereeCanFinishRankedMatchWhilePlayersCannotUseFinishActions(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	if err := svcCtx.DB.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("prepare users: %v", err)
	}
	opponentID := int64(2002)
	refereeID := int64(3003)
	seedFinishReputationUsers(t, svcCtx,
		model.User{Id: 1001, Nickname: "选手甲"},
		model.User{Id: opponentID, Nickname: "选手乙"},
		model.User{Id: refereeID, Nickname: "裁判"},
	)
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id: 9005, UserId: 1001, OpponentId: &opponentID, OpponentName: "选手乙", GameType: 3,
		MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic, FinishConfirmationRequired: true, FinishState: model.FinishStateNone,
		RefereeUserId: &refereeID, Status: 1, MyScore: 2, OpponentScore: 2, CurrentFrameStarted: true, MatchTime: time.Now(),
	}); err != nil {
		t.Fatalf("create referee match: %v", err)
	}

	playerRequest, err := NewRequestFinishMatchLogic(finishReputationCtx(1001), svcCtx).RequestFinishMatch(&types.FinishMatchActionReq{
		MatchId: 9005, ClientActionId: "finish-request-referee-player", BaseRevision: 0,
	})
	if err != nil || playerRequest.Success {
		t.Fatalf("player should not request after referee takeover: resp=%#v err=%v", playerRequest, err)
	}
	playerFinish, err := NewFinishMatchLogic(finishReputationCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId: 9005, ClientActionId: "finish-referee-player", BaseRevision: 0,
	})
	if err != nil || playerFinish.Success {
		t.Fatalf("player should not finish after referee takeover: resp=%#v err=%v", playerFinish, err)
	}

	refereeFinish, err := NewFinishMatchLogic(finishReputationCtx(refereeID), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId: 9005, ClientActionId: "finish-referee-direct", BaseRevision: 0,
	})
	if err != nil || !refereeFinish.Success {
		t.Fatalf("referee should finish ranked match directly: resp=%#v err=%v", refereeFinish, err)
	}
	stored, err := svcCtx.MatchModel.FindById(9005)
	if err != nil || stored == nil || stored.Status != 2 {
		t.Fatalf("expected referee finish to complete match: stored=%+v err=%v", stored, err)
	}
}

func TestRankedFinishBroadcastCarriesMonotonicRevisionSnapshot(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	opponentID := int64(2002)
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id: 9006, UserId: 1001, OpponentId: &opponentID, OpponentName: "对手", GameType: 3,
		MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic, FinishConfirmationRequired: true, FinishState: model.FinishStateNone,
		Status: 1, CurrentFrameStarted: true, MatchTime: time.Now(),
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}

	previousHub := ws.GlobalHub
	hub := ws.NewHub()
	ws.GlobalHub = hub
	t.Cleanup(func() { ws.GlobalHub = previousHub })

	request, err := NewRequestFinishMatchLogic(finishReputationCtx(1001), svcCtx).RequestFinishMatch(&types.FinishMatchActionReq{
		MatchId: 9006, ClientActionId: "finish-request-9006", BaseRevision: 0,
	})
	if err != nil || !request.Success {
		t.Fatalf("request finish: resp=%#v err=%v", request, err)
	}
	requestMessage := readFinishBroadcast(t, hub)
	if requestMessage.Type != "match_finish_request" {
		t.Fatalf("unexpected request broadcast type: %q", requestMessage.Type)
	}
	requestData := requestMessage.Data.(map[string]interface{})
	if requestData["server_revision"].(float64) != 1 || requestData["finish_state"] != model.FinishStatePendingConfirmation {
		t.Fatalf("unexpected request broadcast data: %#v", requestData)
	}

	dispute, err := NewDisputeFinishMatchLogic(finishReputationCtx(opponentID), svcCtx).DisputeFinishMatch(&types.FinishMatchActionReq{
		MatchId: 9006, ClientActionId: "finish-dispute-9006", BaseRevision: request.ServerRevision,
	})
	if err != nil || !dispute.Success {
		t.Fatalf("dispute finish: resp=%#v err=%v", dispute, err)
	}
	disputeMessage := readFinishBroadcast(t, hub)
	if disputeMessage.Type != "match_finish_dispute" {
		t.Fatalf("unexpected dispute broadcast type: %q", disputeMessage.Type)
	}
	disputeData := disputeMessage.Data.(map[string]interface{})
	if disputeData["server_revision"].(float64) != 2 || disputeData["finish_state"] != model.FinishStateNone {
		t.Fatalf("unexpected dispute broadcast data: %#v", disputeData)
	}
}

func TestRankedSettlementBroadcastsRankInfoUpdatedWhenMatchNotificationsAreDisabled(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	opponentID := int64(2002)
	seedFinishReputationUsers(t, svcCtx,
		model.User{Id: 1001, Nickname: "选手甲"},
		model.User{Id: opponentID, Nickname: "选手乙"},
	)
	if err := svcCtx.DB.AutoMigrate(&model.UserNotificationPreference{}); err != nil {
		t.Fatalf("prepare notification preferences: %v", err)
	}
	svcCtx.UserNotificationPreferenceModel = model.NewUserNotificationPreferenceModel(svcCtx.DB)
	for _, userID := range []int64{1001, opponentID} {
		if err := svcCtx.UserNotificationPreferenceModel.Upsert(&model.UserNotificationPreference{
			UserId:               userID,
			MatchResultEnabled:   false,
			FriendRequestEnabled: true,
			ChallengeEnabled:     true,
			TournamentEnabled:    true,
			FollowEnabled:        true,
		}); err != nil {
			t.Fatalf("disable match notification for user %d: %v", userID, err)
		}
	}
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id: 9016, UserId: 1001, OpponentId: &opponentID, OpponentName: "对手", GameType: 3,
		MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic,
		Status: 1, MyScore: 3, OpponentScore: 1, CurrentFrameStarted: true, MatchTime: time.Now(),
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}

	previousHub := ws.GlobalHub
	hub := ws.NewHub()
	ws.GlobalHub = hub
	t.Cleanup(func() { ws.GlobalHub = previousHub })

	resp, err := NewFinishMatchLogic(finishReputationCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId: 9016, ClientActionId: "finish-rank-info-updated-9016", BaseRevision: 0,
	})
	if err != nil || !resp.Success {
		t.Fatalf("finish match: resp=%#v err=%v", resp, err)
	}

	received := make(map[int64]bool)
	deadline := time.After(time.Second)
	for len(received) < 2 {
		select {
		case userMessage := <-hub.SendUser:
			var message ws.Message
			if err := json.Unmarshal(userMessage.Message, &message); err != nil {
				t.Fatalf("decode user message: %v", err)
			}
			if message.Type != "rank_info_updated" {
				continue
			}
			data, ok := message.Data.(map[string]interface{})
			if !ok || data["match_id"] != float64(9016) || data["game_type"] != float64(3) {
				t.Fatalf("unexpected rank update payload: %#v", message.Data)
			}
			received[userMessage.UserId] = true
		case <-deadline:
			t.Fatalf("expected rank updates for both players, received=%#v", received)
		}
	}
	if !received[1001] || !received[opponentID] {
		t.Fatalf("rank updates should reach both players, received=%#v", received)
	}
}

func TestRankedFinishBroadcastUsesViewerSpecificSnapshots(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	opponentID := int64(2002)
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id: 9012, UserId: 1001, OpponentId: &opponentID, OpponentName: "对手", GameType: 3,
		MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic, FinishConfirmationRequired: true, FinishState: model.FinishStateNone,
		Status: 1, MyScore: 5, OpponentScore: 2, CurrentFrameStarted: true, MatchTime: time.Now(),
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}
	previousHub := ws.GlobalHub
	hub := ws.NewHub()
	ws.GlobalHub = hub
	t.Cleanup(func() { ws.GlobalHub = previousHub })

	request, err := NewRequestFinishMatchLogic(finishReputationCtx(1001), svcCtx).RequestFinishMatch(&types.FinishMatchActionReq{
		MatchId: 9012, ClientActionId: "finish-request-viewers-9012", BaseRevision: 0,
	})
	if err != nil || !request.Success {
		t.Fatalf("request finish: resp=%#v err=%v", request, err)
	}
	var broadcast *ws.BroadcastMessage
	select {
	case broadcast = <-hub.Broadcast:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for viewer-specific finish broadcast")
	}
	if len(broadcast.ViewerMessages) < 3 {
		t.Fatalf("expected player1, player2 and anonymous snapshots, got %d", len(broadcast.ViewerMessages))
	}
	var player1Message, player2Message ws.Message
	if err := json.Unmarshal(broadcast.ViewerMessages[1001], &player1Message); err != nil {
		t.Fatalf("decode player1 finish broadcast: %v", err)
	}
	if err := json.Unmarshal(broadcast.ViewerMessages[opponentID], &player2Message); err != nil {
		t.Fatalf("decode player2 finish broadcast: %v", err)
	}
	player1Data := player1Message.Data.(map[string]interface{})
	player2Data := player2Message.Data.(map[string]interface{})
	player1Snapshot := player1Data["snapshot"].(map[string]interface{})
	player2Snapshot := player2Data["snapshot"].(map[string]interface{})
	if player1Snapshot["viewer_role"] != "player1" || player1Snapshot["my_score"].(float64) != 5 || player1Snapshot["can_withdraw_finish"].(bool) != true {
		t.Fatalf("unexpected player1 snapshot: %#v", player1Snapshot)
	}
	if player2Snapshot["viewer_role"] != "player2" || player2Snapshot["my_score"].(float64) != 2 || player2Snapshot["can_confirm_finish"].(bool) != true {
		t.Fatalf("unexpected player2 snapshot: %#v", player2Snapshot)
	}
}

func TestLegacyRankedFinishCompletesAndConfirmedRankedRejectsOldEndpoint(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	opponentID := int64(2002)
	seedFinishReputationUsers(t, svcCtx,
		model.User{Id: 1001, Nickname: "选手甲"},
		model.User{Id: opponentID, Nickname: "选手乙"},
	)

	for _, match := range []model.Match{
		{Id: 9014, UserId: 1001, OpponentId: &opponentID, OpponentName: "选手乙", GameType: 3, MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic, FinishState: model.FinishStateNone, Status: 1, MyScore: 2, OpponentScore: 1, CurrentFrameStarted: true, MatchTime: time.Now()},
		{Id: 9015, UserId: 1001, OpponentId: &opponentID, OpponentName: "选手乙", GameType: 3, MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic, FinishConfirmationRequired: true, FinishState: model.FinishStateNone, Status: 1, MyScore: 2, OpponentScore: 1, CurrentFrameStarted: true, MatchTime: time.Now()},
	} {
		if err := svcCtx.MatchModel.Create(&match); err != nil {
			t.Fatalf("create match %d: %v", match.Id, err)
		}
	}

	legacyResp, err := NewFinishMatchLogic(finishReputationCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{MatchId: 9014, ClientActionId: "legacy-finish-9014", BaseRevision: 0})
	if err != nil || !legacyResp.Success {
		t.Fatalf("legacy ranked match should finish immediately: resp=%#v err=%v", legacyResp, err)
	}
	legacyStored, _ := svcCtx.MatchModel.FindById(9014)
	if legacyStored == nil || legacyStored.Status != 2 {
		t.Fatalf("legacy ranked match was not completed: %+v", legacyStored)
	}

	confirmedResp, err := NewFinishMatchLogic(finishReputationCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{MatchId: 9015, ClientActionId: "old-finish-9015", BaseRevision: 0})
	if err != nil || confirmedResp.Success || confirmedResp.Message == "" || !confirmedResp.Snapshot.CanRequestFinish {
		t.Fatalf("confirmed ranked match should reject old finish endpoint: resp=%#v err=%v", confirmedResp, err)
	}
	confirmedStored, _ := svcCtx.MatchModel.FindById(9015)
	if confirmedStored == nil || confirmedStored.Status != 1 || confirmedStored.FinishState != model.FinishStateNone || confirmedStored.SyncRevision != 0 {
		t.Fatalf("old finish endpoint mutated confirmed ranked match: %+v", confirmedStored)
	}
}

func TestFinishRequestNotificationTargetsTheOtherPlayer(t *testing.T) {
	tests := []struct {
		name        string
		requesterID int64
		recipientID int64
	}{
		{name: "player1 requests", requesterID: 1001, recipientID: 2002},
		{name: "player2 requests", requesterID: 2002, recipientID: 1001},
	}

	for index, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcCtx := newFinishMatchReputationTestSvc(t)
			opponentID := int64(2002)
			seedFinishReputationUsers(t, svcCtx,
				model.User{Id: 1001, Nickname: "选手甲"},
				model.User{Id: opponentID, Nickname: "选手乙"},
			)
			matchID := int64(9020 + index)
			if err := svcCtx.MatchModel.Create(&model.Match{
				Id: matchID, UserId: 1001, OpponentId: &opponentID, OpponentName: "选手乙", GameType: 3,
				MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic, FinishConfirmationRequired: true,
				FinishState: model.FinishStateNone, Status: 1, CurrentFrameStarted: true, MatchTime: time.Now(),
			}); err != nil {
				t.Fatalf("create match: %v", err)
			}
			resp, err := NewRequestFinishMatchLogic(finishReputationCtx(tt.requesterID), svcCtx).RequestFinishMatch(&types.FinishMatchActionReq{
				MatchId: matchID, ClientActionId: "notify-other-player", BaseRevision: 0,
			})
			if err != nil || !resp.Success {
				t.Fatalf("request finish: resp=%#v err=%v", resp, err)
			}
			var notifications []model.Notification
			if err := svcCtx.DB.Where("type = ?", "match_finish_confirmation").Find(&notifications).Error; err != nil {
				t.Fatalf("load notifications: %v", err)
			}
			if len(notifications) != 1 || notifications[0].UserId != tt.recipientID {
				t.Fatalf("expected only user %d to be notified, got %#v", tt.recipientID, notifications)
			}
		})
	}
}

func readFinishBroadcast(t *testing.T, hub *ws.Hub) ws.Message {
	t.Helper()
	select {
	case raw := <-hub.Broadcast:
		var message ws.Message
		if err := json.Unmarshal(raw.Message, &message); err != nil {
			t.Fatalf("decode broadcast: %v", err)
		}
		return message
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for finish broadcast")
		return ws.Message{}
	}
}
