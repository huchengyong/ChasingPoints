package match

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg/ws"
	"chasing_points/internal/testsupport"
	"chasing_points/internal/types"

	"gorm.io/gorm"
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
	requestedAt := time.Now().Add(-5 * time.Second)
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
	wantExpiresAt := requestedAt.Add(model.FinishRequestTTL).Format(time.RFC3339)
	if current.Match.FinishRequestExpiresAt != wantExpiresAt || current.Match.FinishRequestRemainingSeconds < 24 || current.Match.FinishRequestRemainingSeconds > 25 {
		t.Fatalf("current countdown = %q/%d, want %q and about 25s", current.Match.FinishRequestExpiresAt, current.Match.FinishRequestRemainingSeconds, wantExpiresAt)
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
	if detail.Match.FinishRequestExpiresAt != wantExpiresAt || detail.Match.FinishRequestRemainingSeconds < 24 || detail.Match.FinishRequestRemainingSeconds > 25 {
		t.Fatalf("detail countdown = %q/%d, want %q and about 25s", detail.Match.FinishRequestExpiresAt, detail.Match.FinishRequestRemainingSeconds, wantExpiresAt)
	}
}

func TestExpiredFinishRequestReadIsPureUntilWorkerCompletesMatch(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	opponentID := int64(2002)
	requesterID := int64(1001)
	seedFinishReputationUsers(t, svcCtx,
		model.User{Id: requesterID, Nickname: "发起人"},
		model.User{Id: opponentID, Nickname: "对手"},
	)
	requestedAt := time.Now().Add(-model.FinishRequestTTL - time.Second)
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
		MyScore:                    5,
		OpponentScore:              3,
		SyncRevision:               6,
		CurrentFrameStarted:        true,
		MatchTime:                  time.Now().Add(-time.Hour),
	}); err != nil {
		t.Fatalf("create expired pending match: %v", err)
	}
	recorder := testsupport.NewSQLWriteRecorder()
	readDB := svcCtx.DB.Session(&gorm.Session{Logger: recorder})
	svcCtx.MatchModel = model.NewMatchModel(readDB)
	svcCtx.UserModel = model.NewUserModel(readDB)
	previousHub := ws.GlobalHub
	hub := ws.NewHub()
	ws.GlobalHub = hub
	t.Cleanup(func() { ws.GlobalHub = previousHub })

	// 到期未处理：读取仍如实返回待确认、剩余0秒且无任何操作权限，不被改写成已取消。
	for _, viewer := range []int64{requesterID, opponentID} {
		current, err := NewGetCurrentMatchLogic(finishReputationCtx(viewer), svcCtx).GetCurrentMatch()
		if err != nil || !current.Success || current.Match == nil {
			t.Fatalf("expired current match for %d: resp=%#v err=%v", viewer, current, err)
		}
		m := current.Match
		if m.FinishState != model.FinishStatePendingConfirmation || m.ServerRevision != 6 || m.FinishRequestRemainingSeconds != 0 || m.FinishRequestExpiresAt == "" {
			t.Fatalf("expired read must stay pending with 0 seconds left, got %+v", m)
		}
		if m.CanWithdrawFinish || m.CanConfirmFinish || m.CanDisputeFinish || m.CanScore {
			t.Fatalf("expired read must not offer finish actions, got %+v", m)
		}
	}
	if writes := recorder.Writes(); len(writes) != 0 {
		t.Fatalf("current match GET must not issue writes: %q", writes)
	}

	if err := NewFinishRequestExpiryWorker(svcCtx).RunOnce(context.Background()); err != nil {
		t.Fatalf("expire finish request worker: %v", err)
	}
	timeoutMessage := readFinishBroadcast(t, hub)
	if timeoutMessage.Type != "match_finish_timeout" {
		t.Fatalf("unexpected timeout broadcast type: %q", timeoutMessage.Type)
	}
	stored, err := svcCtx.MatchModel.FindById(9011)
	if err != nil || stored == nil || stored.Status != 2 || stored.Result == nil || *stored.Result != 1 || stored.FinishState != model.FinishStateNone {
		t.Fatalf("worker must complete the match with the frozen score: stored=%+v err=%v", stored, err)
	}
	if stored.CompletionSource != model.CompletionSourceFinishTimeout || stored.CompletedByUserId == nil || *stored.CompletedByUserId != requesterID {
		t.Fatalf("timeout completion attribution = %q/%v, want finish_timeout by requester", stored.CompletionSource, stored.CompletedByUserId)
	}
	var timeoutActions, rankLogs int64
	svcCtx.DB.Model(&model.MatchAction{}).Where("match_id = ? AND action_type = ?", 9011, finishTimeoutActionType).Count(&timeoutActions)
	svcCtx.DB.Model(&model.RankChangeLog{}).Where("match_id = ?", 9011).Count(&rankLogs)
	if timeoutActions != 1 || rankLogs != 2 {
		t.Fatalf("timeout must write one finish_timeout action and settle rank for both players immediately, actions=%d rankLogs=%d", timeoutActions, rankLogs)
	}

	// 重复运行不重复结算。
	if err := NewFinishRequestExpiryWorker(svcCtx).RunOnce(context.Background()); err != nil {
		t.Fatalf("second worker run: %v", err)
	}
	svcCtx.DB.Model(&model.RankChangeLog{}).Where("match_id = ?", 9011).Count(&rankLogs)
	if rankLogs != 2 {
		t.Fatalf("second worker run must not settle again, rankLogs=%d", rankLogs)
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
