package ws

import (
	"encoding/json"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBuildMatchSyncDataForViewerIncludesFinishCapabilities(t *testing.T) {
	requesterID := int64(1001)
	match := &model.Match{
		Id:                         99,
		UserId:                     requesterID,
		OpponentId:                 hubInt64Ptr(2002),
		Status:                     1,
		MatchMode:                  model.MatchModeRanked,
		FinishConfirmationRequired: true,
		Visibility:                 model.MatchVisibilityPublic,
		FinishState:                model.FinishStatePendingConfirmation,
		FinishRequestedBy:          &requesterID,
		SyncRevision:               7,
		MyScore:                    4,
		OpponentScore:              3,
		CurrentFrameStarted:        true,
	}
	data := buildMatchSyncDataForViewer(match, 2002, 0, model.SnookerRoundState{}, nil)
	if data.FinishState != model.FinishStatePendingConfirmation || !data.CanConfirmFinish || !data.CanDisputeFinish || data.CanScore {
		t.Fatalf("unexpected pending viewer capabilities: %+v", data)
	}
	if data.ViewerRole != "player2" || data.MyScore != 3 || data.OpponentScore != 4 {
		t.Fatalf("unexpected viewer perspective: %+v", data)
	}
}

func TestResolveMatchSyncCapabilitiesCoversLegacyRankedConfirmedRankedAndRefereePending(t *testing.T) {
	opponentID := int64(2002)
	legacy := &model.Match{UserId: 1001, OpponentId: &opponentID, MatchMode: model.MatchModeRanked, Status: 1}
	_, _, _, _, _, canFinish, canRequest, _, _, _ := resolveMatchSyncCapabilities(legacy, 1001)
	if !canFinish || canRequest {
		t.Fatalf("unexpected legacy ranked capabilities: finish=%v request=%v", canFinish, canRequest)
	}

	confirmed := &model.Match{UserId: 1001, OpponentId: &opponentID, MatchMode: model.MatchModeRanked, FinishConfirmationRequired: true, Status: 1}
	_, _, _, _, _, canFinish, canRequest, _, _, _ = resolveMatchSyncCapabilities(confirmed, 1001)
	if canFinish || !canRequest {
		t.Fatalf("unexpected confirmed ranked capabilities: finish=%v request=%v", canFinish, canRequest)
	}

	refereeID := int64(3003)
	confirmed.RefereeUserId = &refereeID
	confirmed.FinishState = model.FinishStatePendingConfirmation
	confirmed.FinishRequestedBy = hubInt64Ptr(1001)
	_, _, _, _, _, canFinish, canRequest, canConfirm, canDispute, canWithdraw := resolveMatchSyncCapabilities(confirmed, opponentID)
	if canFinish || canRequest || canConfirm || canDispute || canWithdraw {
		t.Fatal("referee-bound pending match leaked player finish capabilities")
	}
}

func TestSendMatchSyncBroadcastsExpiredFinishOnceForAllViewers(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.Match{}, &model.MatchRound{}, &model.MatchAction{}); err != nil {
		t.Fatalf("prepare ws schema: %v", err)
	}
	svcCtx := &svc.ServiceContext{DB: db, MatchModel: model.NewMatchModel(db)}
	requesterID := int64(1001)
	opponentID := int64(2002)
	requestedAt := time.Now().Add(-25 * time.Hour)
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id: 199, UserId: requesterID, OpponentId: &opponentID, OpponentName: "对手", GameType: 3,
		MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic, FinishConfirmationRequired: true,
		FinishState: model.FinishStatePendingConfirmation, FinishRequestedBy: &requesterID,
		FinishRequestedAt: &requestedAt, FinishRequestRevision: 4, SyncRevision: 4,
		Status: 1, CurrentFrameStarted: true, MatchTime: time.Now().Add(-26 * time.Hour),
	}); err != nil {
		t.Fatalf("create expired match: %v", err)
	}

	hub := NewHub()
	first := &Client{Hub: hub, SvcCtx: svcCtx, MatchId: 199, UserId: requesterID, Send: make(chan []byte, 4)}
	second := &Client{Hub: hub, SvcCtx: svcCtx, MatchId: 199, UserId: opponentID, Send: make(chan []byte, 4)}
	first.sendMatchSync()

	var broadcast *BroadcastMessage
	select {
	case broadcast = <-hub.Broadcast:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for expiry broadcast")
	}
	if len(broadcast.ViewerMessages) < 3 {
		t.Fatalf("expected anonymous and both player snapshots, got %d", len(broadcast.ViewerMessages))
	}
	var opponentMessage Message
	if err := json.Unmarshal(broadcast.ViewerMessages[opponentID], &opponentMessage); err != nil {
		t.Fatalf("decode opponent expiry broadcast: %v", err)
	}
	if opponentMessage.Type != "match_finish_expired" {
		t.Fatalf("unexpected expiry message: %#v", opponentMessage)
	}
	snapshot := opponentMessage.Data.(map[string]interface{})["snapshot"].(map[string]interface{})
	if snapshot["finish_state"] != model.FinishStateNone || snapshot["server_revision"].(float64) != 5 || snapshot["can_score"].(bool) != true {
		t.Fatalf("unexpected restored snapshot: %#v", snapshot)
	}

	second.sendMatchSync()
	select {
	case duplicate := <-hub.Broadcast:
		t.Fatalf("unexpected duplicate expiry broadcast: %#v", duplicate)
	case <-time.After(30 * time.Millisecond):
	}
	var actionCount int64
	if err := db.Model(&model.MatchAction{}).Where("match_id = ? AND action_type = ?", 199, "finish_expired").Count(&actionCount).Error; err != nil {
		t.Fatalf("count expiry actions: %v", err)
	}
	if actionCount != 1 {
		t.Fatalf("expected one finish_expired action, got %d", actionCount)
	}
}

func hubInt64Ptr(value int64) *int64 {
	return &value
}

func TestBroadcastToMatchForUsersRoutesViewerSnapshots(t *testing.T) {
	hub := NewHub()
	request := &Message{Type: "match_finish_request", Data: map[string]interface{}{"snapshot": "player1"}}
	opponent := &Message{Type: "match_finish_request", Data: map[string]interface{}{"snapshot": "player2"}}
	hub.BroadcastToMatchForUsers(99, map[int64]*Message{1001: request, 2002: opponent})
	broadcast := <-hub.Broadcast
	if len(broadcast.ViewerMessages) != 2 {
		t.Fatalf("expected two viewer payloads, got %d", len(broadcast.ViewerMessages))
	}
	var decoded Message
	if err := json.Unmarshal(broadcast.ViewerMessages[2002], &decoded); err != nil {
		t.Fatalf("decode viewer payload: %v", err)
	}
	if decoded.Data.(map[string]interface{})["snapshot"] != "player2" {
		t.Fatalf("expected player2 payload, got %#v", decoded.Data)
	}
}

func TestBuildMatchSyncDataIncludesServerRevision(t *testing.T) {
	match := &model.Match{
		Id:                        36,
		MyScore:                   4,
		OpponentScore:             2,
		CurrentFrameMyScore:       18,
		CurrentFrameOpponentScore: 7,
		CurrentFrameStarted:       true,
		Status:                    1,
		SyncRevision:              9,
	}

	data := buildMatchSyncData(match, 1, model.SnookerRoundState{}, nil)
	if data.ServerRevision != 9 {
		t.Fatalf("expected sync data server revision 9, got %d", data.ServerRevision)
	}
}
