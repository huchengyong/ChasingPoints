package match

import (
	"context"
	"errors"
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
)

func TestAchievementSyncWorkerProcessesFixedBatchesIdempotently(t *testing.T) {
	svcCtx := newMatchAchievementClosedLoopTestSvc(t)
	result := 3
	matches := make([]model.Match, 0, achievementSyncWorkerBatchSize+1)
	for id := int64(1); id <= achievementSyncWorkerBatchSize+1; id++ {
		matches = append(matches, model.Match{
			Id: id, UserId: 1, GameType: 3, MatchMode: model.MatchModePractice, Status: 2, Result: &result,
		})
	}
	if err := svcCtx.DB.Create(&matches).Error; err != nil {
		t.Fatalf("seed completed matches: %v", err)
	}
	worker := NewAchievementSyncWorker(svcCtx)
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("first sync batch: %v", err)
	}
	assertAchievementSyncedCount(t, svcCtx, achievementSyncWorkerBatchSize)
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("second sync batch: %v", err)
	}
	assertAchievementSyncedCount(t, svcCtx, achievementSyncWorkerBatchSize+1)
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("idempotent sync batch: %v", err)
	}
	assertAchievementSyncedCount(t, svcCtx, achievementSyncWorkerBatchSize+1)
}

func TestAchievementSyncWorkerStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := NewAchievementSyncWorker(&svc.ServiceContext{}).RunOnce(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled worker run error=%v, want context.Canceled", err)
	}
}

func assertAchievementSyncedCount(t *testing.T, svcCtx *svc.ServiceContext, want int) {
	t.Helper()
	var count int64
	if err := svcCtx.DB.Model(&model.Match{}).Where("achievement_synced_at IS NOT NULL").Count(&count).Error; err != nil || count != int64(want) {
		t.Fatalf("synced match count=%d want=%d err=%v", count, want, err)
	}
}
