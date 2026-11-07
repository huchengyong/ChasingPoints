package logic

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

func TestChallengeExpiryWorkerProcessesFixedBatchesIdempotently(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Challenge{}); err != nil {
		t.Fatalf("prepare challenge schema: %v", err)
	}
	now := time.Now()
	challenges := make([]model.Challenge, 0, challengeExpiryWorkerBatchSize+2)
	for i := 1; i <= challengeExpiryWorkerBatchSize+1; i++ {
		challenges = append(challenges, model.Challenge{Id: int64(i), FromUserId: 2, ToUserId: 1, Status: 0, ExpiresAt: now.Add(-time.Hour)})
	}
	challenges = append(challenges, model.Challenge{Id: 1000, FromUserId: 2, ToUserId: 1, Status: 0, ExpiresAt: now.Add(time.Hour)})
	if err := db.Create(&challenges).Error; err != nil {
		t.Fatalf("seed challenges: %v", err)
	}
	worker := NewChallengeExpiryWorker(&svc.ServiceContext{ChallengeModel: model.NewChallengeModel(db)})
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("first expiry batch: %v", err)
	}
	assertChallengeStatusCount(t, db, 3, challengeExpiryWorkerBatchSize)
	assertChallengeStatusCount(t, db, 0, 2)
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("second expiry batch: %v", err)
	}
	assertChallengeStatusCount(t, db, 3, challengeExpiryWorkerBatchSize+1)
	assertChallengeStatusCount(t, db, 0, 1)
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("idempotent expiry batch: %v", err)
	}
	assertChallengeStatusCount(t, db, 3, challengeExpiryWorkerBatchSize+1)
}

func TestChallengeExpiryWorkerStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := NewChallengeExpiryWorker(&svc.ServiceContext{}).RunOnce(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled worker run error=%v, want context.Canceled", err)
	}
}

func assertChallengeStatusCount(t *testing.T, db *gorm.DB, status int, want int) {
	t.Helper()
	var count int64
	if err := db.Model(&model.Challenge{}).Where("status = ?", status).Count(&count).Error; err != nil || count != int64(want) {
		t.Fatalf("challenge status %d count=%d want=%d err=%v", status, count, want, err)
	}
}
