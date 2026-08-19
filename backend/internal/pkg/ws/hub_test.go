package ws

import (
	"encoding/json"
	"strings"
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

func TestResolveMatchSyncCapabilitiesAllowsEligibleSnookerRefereeFinish(t *testing.T) {
	refereeID := int64(3003)
	opponentID := int64(2002)
	match := &model.Match{
		UserId:              1001,
		OpponentId:          &opponentID,
		RefereeUserId:       &refereeID,
		GameType:            1,
		SnookerRulesVersion: model.SnookerRulesVersionWPBSA,
		SnookerFormat:       model.SnookerFormatFree,
		MyScore:             1,
		Status:              1,
	}

	role, _, _, _, _, canFinish, _, _, _, _ := resolveMatchSyncCapabilities(match, refereeID)
	if role != "referee" || !canFinish {
		t.Fatalf("eligible snooker referee should be able to finish: role=%q canFinish=%v", role, canFinish)
	}

	match.CurrentFrameStarted = true
	_, _, _, _, _, canFinish, _, _, _, _ = resolveMatchSyncCapabilities(match, refereeID)
	if canFinish {
		t.Fatal("snooker referee must not finish while the current frame is active")
	}
}

func TestMatchWebSocketPayloadKeepsFalseFormatCapabilities(t *testing.T) {
	for name, payload := range map[string]interface{}{
		"score update": ScoreUpdateData{},
		"sync":         MatchSyncData{},
	} {
		t.Run(name, func(t *testing.T) {
			encoded, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("marshal payload: %v", err)
			}
			if !strings.Contains(string(encoded), `"can_change_snooker_format":false`) {
				t.Fatalf("false format capability must be explicit: %s", encoded)
			}
			if !strings.Contains(string(encoded), `"can_change_match_format":false`) {
				t.Fatalf("false pool format capability must be explicit: %s", encoded)
			}
		})
	}
}

func TestSendMatchSyncReturnsEffectiveExpiredFinishViewWithoutWriting(t *testing.T) {
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

	var syncMessage Message
	select {
	case payload := <-first.Send:
		if err := json.Unmarshal(payload, &syncMessage); err != nil {
			t.Fatalf("decode effective sync: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for sync message")
	}
	snapshot := syncMessage.Data.(map[string]interface{})
	if syncMessage.Type != "sync" || snapshot["finish_state"] != model.FinishStateNone || snapshot["server_revision"].(float64) != 4 || snapshot["can_score"].(bool) != true {
		t.Fatalf("unexpected effective sync snapshot: %#v", syncMessage)
	}

	second.sendMatchSync()
	select {
	case unexpected := <-hub.Broadcast:
		t.Fatalf("sync must not persist or broadcast expiration: %#v", unexpected)
	case <-time.After(30 * time.Millisecond):
	}
	var actionCount int64
	if err := db.Model(&model.MatchAction{}).Where("match_id = ? AND action_type = ?", 199, "finish_expired").Count(&actionCount).Error; err != nil {
		t.Fatalf("count expiry actions: %v", err)
	}
	if actionCount != 0 {
		t.Fatalf("sync must not create expiry actions, got %d", actionCount)
	}
	stored, err := svcCtx.MatchModel.FindById(199)
	if err != nil || stored == nil || stored.FinishState != model.FinishStatePendingConfirmation || stored.SyncRevision != 4 {
		t.Fatalf("sync must leave stored finish request unchanged: %+v err=%v", stored, err)
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

func TestSendMatchSyncIncludesRefereeIdentityAndCompletionAttribution(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}, &model.MatchRound{}, &model.MatchAction{}); err != nil {
		t.Fatalf("prepare ws schema: %v", err)
	}
	refereeID := int64(3003)
	opponentID := int64(2002)
	joinedAt := time.Now().Add(-10 * time.Minute)
	endTime := time.Now()
	if err := db.Create(&model.User{Id: refereeID, Nickname: "裁判丙", Avatar: "referee.png", Status: 1}).Error; err != nil {
		t.Fatalf("create referee: %v", err)
	}
	if err := db.Create(&model.Match{
		Id: 201, UserId: 1001, OpponentId: &opponentID, OpponentName: "选手乙", GameType: 3,
		Status: 2, RefereeUserId: &refereeID, RefereeJoinedAt: &joinedAt, EndTime: &endTime,
		CompletedByUserId: &refereeID, CompletionSource: model.CompletionSourceReferee,
		CurrentFrameStarted: false, MatchTime: joinedAt.Add(-20 * time.Minute),
	}).Error; err != nil {
		t.Fatalf("create match: %v", err)
	}
	svcCtx := &svc.ServiceContext{
		DB:         db,
		UserModel:  model.NewUserModel(db),
		MatchModel: model.NewMatchModel(db),
	}
	client := &Client{Hub: NewHub(), SvcCtx: svcCtx, MatchId: 201, UserId: 1001, Send: make(chan []byte, 1)}
	client.sendMatchSync()

	var message Message
	select {
	case payload := <-client.Send:
		if err := json.Unmarshal(payload, &message); err != nil {
			t.Fatalf("decode sync: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for sync payload")
	}
	data := message.Data.(map[string]interface{})
	if data["referee_name"] != "裁判丙" || data["referee_avatar"] != "referee.png" || data["referee_joined_at"] == "" {
		t.Fatalf("missing referee identity in sync: %#v", data)
	}
	if data["completed_by_user_id"].(float64) != float64(refereeID) || data["completion_source"] != model.CompletionSourceReferee {
		t.Fatalf("missing completion attribution in sync: %#v", data)
	}
}
