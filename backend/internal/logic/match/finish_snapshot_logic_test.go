package match

import (
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg/ws"
	"chasing_points/internal/types"
)

func TestPendingFinishIsReturnedByCurrentAndDetailWithViewerCapabilities(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	if err := svcCtx.DB.AutoMigrate(&model.MatchAchievement{}, &model.Opponent{}); err != nil {
		t.Fatalf("prepare detail schema: %v", err)
	}
	opponentID := int64(2002)
	requesterID := int64(1001)
	seedFinishReputationUsers(t, svcCtx,
		model.User{Id: requesterID, Nickname: "发起人"},
		model.User{Id: opponentID, Nickname: "对手"},
	)
	requestedAt := time.Now().Add(-time.Hour)
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id:                         9010,
		UserId:                     requesterID,
		OpponentId:                 &opponentID,
		OpponentName:               "对手",
		GameType:                   3,
		MatchMode:                  model.MatchModeRanked,
		FinishConfirmationRequired: true,
		Visibility:                 model.MatchVisibilityPublic,
		FinishState:                model.FinishStatePendingConfirmation,
		FinishRequestedBy:          &requesterID,
		FinishRequestedAt:          &requestedAt,
		FinishRequestRevision:      4,
		Status:                     1,
		MyScore:                    5,
		OpponentScore:              3,
		CurrentFrameStarted:        true,
		SyncRevision:               4,
		MatchTime:                  time.Now().Add(-2 * time.Hour),
	}); err != nil {
		t.Fatalf("create pending match: %v", err)
	}

	current, err := NewGetCurrentMatchLogic(finishReputationCtx(opponentID), svcCtx).GetCurrentMatch()
	if err != nil || !current.Success || current.Match == nil {
		t.Fatalf("current pending match: resp=%#v err=%v", current, err)
	}
	if current.Match.FinishState != model.FinishStatePendingConfirmation || current.Match.FinishRequestedBy != requesterID {
		t.Fatalf("unexpected current pending state: %+v", current.Match)
	}
	if current.Match.ViewerRole != "player2" || !current.Match.CanConfirmFinish || !current.Match.CanDisputeFinish || current.Match.CanScore {
		t.Fatalf("unexpected current capabilities: %+v", current.Match)
	}

	detail, err := NewGetMatchDetailLogic(finishReputationCtx(opponentID), svcCtx).GetMatchDetail(&types.GetMatchDetailReq{MatchId: 9010})
	if err != nil || !detail.Success {
		t.Fatalf("detail pending match: resp=%#v err=%v", detail, err)
	}
	if detail.Match.FinishState != model.FinishStatePendingConfirmation || detail.Match.FinishRequestedBy != requesterID {
		t.Fatalf("unexpected detail pending state: %+v", detail.Match)
	}
	if detail.Match.ViewerRole != "player2" || !detail.Match.CanConfirmFinish || !detail.Match.CanDisputeFinish || detail.Match.CanScore {
		t.Fatalf("unexpected detail capabilities: %+v", detail.Match)
	}
}

func TestExpiredFinishRequestRestoresCurrentMatchAndBumpsRevision(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	opponentID := int64(2002)
	requesterID := int64(1001)
	requestedAt := time.Now().Add(-25 * time.Hour)
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id:                         9011,
		UserId:                     requesterID,
		OpponentId:                 &opponentID,
		OpponentName:               "对手",
		GameType:                   3,
		MatchMode:                  model.MatchModeRanked,
		FinishConfirmationRequired: true,
		Visibility:                 model.MatchVisibilityPublic,
		FinishState:                model.FinishStatePendingConfirmation,
		FinishRequestedBy:          &requesterID,
		FinishRequestedAt:          &requestedAt,
		FinishRequestRevision:      6,
		Status:                     1,
		SyncRevision:               6,
		CurrentFrameStarted:        true,
		MatchTime:                  time.Now().Add(-26 * time.Hour),
	}); err != nil {
		t.Fatalf("create expired pending match: %v", err)
	}
	previousHub := ws.GlobalHub
	hub := ws.NewHub()
	ws.GlobalHub = hub
	t.Cleanup(func() { ws.GlobalHub = previousHub })

	current, err := NewGetCurrentMatchLogic(finishReputationCtx(requesterID), svcCtx).GetCurrentMatch()
	if err != nil || !current.Success || current.Match == nil {
		t.Fatalf("expired current match should be restored: resp=%#v err=%v", current, err)
	}
	if current.Match.FinishState != model.FinishStateNone || current.Match.ServerRevision != 7 {
		t.Fatalf("expected restored current match at revision 7, got %+v", current.Match)
	}
	expiredMessage := readFinishBroadcast(t, hub)
	if expiredMessage.Type != "match_finish_expired" {
		t.Fatalf("unexpected expiry broadcast type: %q", expiredMessage.Type)
	}
	expiredData := expiredMessage.Data.(map[string]interface{})
	if expiredData["server_revision"].(float64) != 7 || expiredData["finish_state"] != model.FinishStateNone {
		t.Fatalf("unexpected expiry broadcast data: %#v", expiredData)
	}
	stored, err := svcCtx.MatchModel.FindById(9011)
	if err != nil || stored == nil || stored.FinishState != model.FinishStateNone || stored.FinishRequestedBy != nil || stored.SyncRevision != 7 {
		t.Fatalf("unexpected stored expired state: stored=%+v err=%v", stored, err)
	}
	var expiredActionCount int64
	if err := svcCtx.DB.Model(&model.MatchAction{}).Where("match_id = ? AND action_type = ?", 9011, "finish_expired").Count(&expiredActionCount).Error; err != nil {
		t.Fatalf("count expired action: %v", err)
	}
	if expiredActionCount != 1 {
		t.Fatalf("expected one finish_expired action, got %d", expiredActionCount)
	}
}

func TestMatchDetailReturnsStableParticipantIDsForPlayersAndReferee(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	if err := svcCtx.DB.AutoMigrate(&model.MatchAchievement{}, &model.Opponent{}); err != nil {
		t.Fatalf("prepare detail schema: %v", err)
	}
	opponentID := int64(2002)
	refereeID := int64(3003)
	joinedAt := time.Now().Add(-10 * time.Minute)
	seedFinishReputationUsers(t, svcCtx,
		model.User{Id: 1001, Nickname: "选手甲"},
		model.User{Id: opponentID, Nickname: "选手乙"},
		model.User{Id: refereeID, Nickname: "裁判"},
	)
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id: 9016, UserId: 1001, OpponentId: &opponentID, OpponentName: "选手乙", GameType: 3,
		MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate, RefereeUserId: &refereeID, RefereeJoinedAt: &joinedAt,
		Status: 1, CurrentFrameStarted: true, MatchTime: time.Now(),
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}

	for _, tt := range []struct {
		name       string
		viewerID   int64
		opponentID int64
	}{
		{name: "player1", viewerID: 1001, opponentID: 2002},
		{name: "player2", viewerID: 2002, opponentID: 1001},
		{name: "referee", viewerID: 3003, opponentID: 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := NewGetMatchDetailLogic(finishReputationCtx(tt.viewerID), svcCtx).GetMatchDetail(&types.GetMatchDetailReq{MatchId: 9016})
			if err != nil || !resp.Success {
				t.Fatalf("get detail: resp=%#v err=%v", resp, err)
			}
			if resp.Match.Player1Id != 1001 || resp.Match.Player2Id != 2002 || resp.Match.OpponentId != tt.opponentID {
				t.Fatalf("unexpected participant ids: %+v", resp.Match)
			}
			if resp.Match.RefereeName != "裁判" || resp.Match.RefereeJoinedAt == "" {
				t.Fatalf("missing referee detail: %+v", resp.Match)
			}
		})
	}
}
