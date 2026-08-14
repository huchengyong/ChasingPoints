package match

import (
	"context"
	"errors"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFinishRequestExpiryWorkerProcessesFixedBatches(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Match{}, &model.MatchAction{}); err != nil {
		t.Fatalf("prepare match schema: %v", err)
	}
	requesterID := int64(1)
	opponentID := int64(2)
	requestedAt := time.Now().Add(-model.FinishRequestTTL - time.Minute)
	matches := make([]model.Match, 0, finishRequestExpiryWorkerBatchSize+1)
	for i := 1; i <= finishRequestExpiryWorkerBatchSize+1; i++ {
		matches = append(matches, model.Match{Id: int64(i), UserId: requesterID, OpponentId: &opponentID, GameType: 3, Status: 1, FinishState: model.FinishStatePendingConfirmation, FinishRequestedBy: &requesterID, FinishRequestedAt: &requestedAt, FinishRequestRevision: 1})
	}
	if err := db.Create(&matches).Error; err != nil {
		t.Fatalf("seed stale finish requests: %v", err)
	}
	worker := NewFinishRequestExpiryWorker(&svc.ServiceContext{MatchModel: model.NewMatchModel(db)})
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("first expiry batch: %v", err)
	}
	assertFinishStateCount(t, db, model.FinishStateNone, finishRequestExpiryWorkerBatchSize)
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("second expiry batch: %v", err)
	}
	assertFinishStateCount(t, db, model.FinishStateNone, finishRequestExpiryWorkerBatchSize+1)
}

func TestFinishRequestExpiryWorkerStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := NewFinishRequestExpiryWorker(&svc.ServiceContext{}).RunOnce(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled worker run error=%v, want context.Canceled", err)
	}
}

func assertFinishStateCount(t *testing.T, db *gorm.DB, state string, want int) {
	t.Helper()
	var count int64
	if err := db.Model(&model.Match{}).Where("finish_state = ?", state).Count(&count).Error; err != nil || count != int64(want) {
		t.Fatalf("finish state %q count=%d want=%d err=%v", state, count, want, err)
	}
}
