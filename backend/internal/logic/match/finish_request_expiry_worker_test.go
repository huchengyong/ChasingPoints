package match

import (
	"context"
	"errors"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/gorm"
)

func TestFinishRequestExpiryWorkerCompletesFixedBatchesAndSkipsBrokenRows(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	requesterID := int64(1)
	opponentID := int64(2)
	seedFinishReputationUsers(t, svcCtx, model.User{Id: requesterID, Nickname: "发起人"}, model.User{Id: opponentID, Nickname: "对手"})
	requestedAt := time.Now().Add(-model.FinishRequestTTL - time.Minute)
	brokenAt := requestedAt.Add(-time.Minute)
	matches := []model.Match{{
		// 缺少发起方的异常数据排在最前，不能挡住其他到期请求。
		Id: 1000, UserId: requesterID, OpponentId: &opponentID, GameType: 3, MatchMode: model.MatchModeRanked,
		FinishConfirmationRequired: true, Status: 1, FinishState: model.FinishStatePendingConfirmation,
		FinishRequestedAt: &brokenAt, FinishRequestRevision: 1, MatchTime: time.Now(),
	}}
	for i := 1; i < finishRequestExpiryWorkerBatchSize+1; i++ {
		matches = append(matches, model.Match{Id: int64(i), UserId: requesterID, OpponentId: &opponentID, GameType: 3, MatchMode: model.MatchModeRanked,
			FinishConfirmationRequired: true, Status: 1, MyScore: 3, OpponentScore: 1, FinishState: model.FinishStatePendingConfirmation,
			FinishRequestedBy: &requesterID, FinishRequestedAt: &requestedAt, FinishRequestRevision: 1, MatchTime: time.Now()})
	}
	if err := svcCtx.DB.Create(&matches).Error; err != nil {
		t.Fatalf("seed stale finish requests: %v", err)
	}
	worker := NewFinishRequestExpiryWorker(svcCtx)
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("first expiry batch: %v", err)
	}
	assertCompletedTimeoutCount(t, svcCtx.DB, finishRequestExpiryWorkerBatchSize-1)
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("second expiry batch: %v", err)
	}
	assertCompletedTimeoutCount(t, svcCtx.DB, finishRequestExpiryWorkerBatchSize)
	var broken model.Match
	if err := svcCtx.DB.First(&broken, 1000).Error; err != nil || broken.Status != 1 {
		t.Fatalf("broken row must stay untouched for investigation: %+v err=%v", broken, err)
	}
}

func TestFinishRequestExpiryWorkerLeavesUnexpiredRequests(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	requesterID := int64(1)
	opponentID := int64(2)
	requestedAt := time.Now().Add(-model.FinishRequestTTL + 2*time.Second)
	if err := svcCtx.DB.Create(&model.Match{Id: 1, UserId: requesterID, OpponentId: &opponentID, GameType: 3, MatchMode: model.MatchModeRanked,
		FinishConfirmationRequired: true, Status: 1, FinishState: model.FinishStatePendingConfirmation,
		FinishRequestedBy: &requesterID, FinishRequestedAt: &requestedAt, FinishRequestRevision: 1, MatchTime: time.Now()}).Error; err != nil {
		t.Fatalf("seed pending request: %v", err)
	}
	if err := NewFinishRequestExpiryWorker(svcCtx).RunOnce(context.Background()); err != nil {
		t.Fatalf("worker run: %v", err)
	}
	var stored model.Match
	if err := svcCtx.DB.First(&stored, 1).Error; err != nil || stored.Status != 1 || stored.FinishState != model.FinishStatePendingConfirmation {
		t.Fatalf("request with 2s left must stay pending: %+v err=%v", stored, err)
	}
}

func TestFinishRequestExpiryWorkerStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := NewFinishRequestExpiryWorker(&svc.ServiceContext{}).RunOnce(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled worker run error=%v, want context.Canceled", err)
	}
}

func assertCompletedTimeoutCount(t *testing.T, db *gorm.DB, want int) {
	t.Helper()
	var count int64
	if err := db.Model(&model.Match{}).Where("status = ? AND completion_source = ?", 2, model.CompletionSourceFinishTimeout).Count(&count).Error; err != nil || count != int64(want) {
		t.Fatalf("timeout-completed matches=%d want=%d err=%v", count, want, err)
	}
}
