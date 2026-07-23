package logic

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"chasing_points/internal/model"
)

type fakeSeasonRolloverRunner struct {
	mu              sync.Mutex
	settlementCalls int
	activationCalls int
	failuresLeft    int
}

func (f *fakeSeasonRolloverRunner) SettleSeasonAt(_ context.Context, seasonId int64, _ time.Time) (*SeasonSettlementSummary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.settlementCalls++
	if f.failuresLeft > 0 {
		f.failuresLeft--
		return nil, errors.New("temporary settlement failure")
	}
	return &SeasonSettlementSummary{SeasonId: seasonId}, nil
}

func (f *fakeSeasonRolloverRunner) ActivateReadySeasonAt(_ context.Context, _ time.Time) (*model.Season, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.activationCalls++
	return nil, nil
}

func TestSeasonRolloverWorkerRetriesFailureOnNextRun(t *testing.T) {
	svcCtx := newSeasonSettlementTestSvc(t)
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	season := model.Season{
		Id:        3,
		Name:      "S3",
		StartDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
		Status:    1,
	}
	if err := svcCtx.SeasonModel.Create(&season); err != nil {
		t.Fatalf("create expired season: %v", err)
	}
	runner := &fakeSeasonRolloverRunner{failuresLeft: 1}
	worker := newSeasonRolloverWorkerWithDeps(svcCtx, runner, func() time.Time { return now }, time.Minute)

	if err := worker.RunOnce(context.Background()); err == nil {
		t.Fatal("expected first worker run to fail")
	}
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("expected second worker run to retry successfully: %v", err)
	}
	if runner.settlementCalls != 2 || runner.activationCalls != 1 {
		t.Fatalf("unexpected runner calls: settlements=%d activations=%d", runner.settlementCalls, runner.activationCalls)
	}
}

func TestSeasonRolloverWorkerSkipsCompletedSettlement(t *testing.T) {
	svcCtx := newSeasonSettlementTestSvc(t)
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	season := model.Season{
		Id:        3,
		Name:      "S3",
		StartDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
		Status:    1,
	}
	if err := svcCtx.SeasonModel.Create(&season); err != nil {
		t.Fatalf("create expired season: %v", err)
	}
	completedAt := now.Add(-time.Minute)
	if err := svcCtx.SeasonSettlementModel.CreateWithTx(nil, &model.SeasonSettlement{
		SeasonId:    season.Id,
		Status:      model.SeasonSettlementStatusCompleted,
		Attempts:    1,
		CompletedAt: &completedAt,
	}); err != nil {
		t.Fatalf("create completed settlement: %v", err)
	}
	runner := &fakeSeasonRolloverRunner{}
	worker := newSeasonRolloverWorkerWithDeps(svcCtx, runner, func() time.Time { return now }, time.Minute)

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("run worker with completed settlement: %v", err)
	}
	if runner.settlementCalls != 0 {
		t.Fatalf("expected completed settlement skipped, got %d calls", runner.settlementCalls)
	}
}

func TestSeasonRolloverWorkersConvergeAcrossInstances(t *testing.T) {
	svcCtx := newSeasonSettlementTestSvc(t)
	season, _, now := seedSeasonSettlementScenario(t, svcCtx, false)
	service1 := NewSeasonSettlementService(svcCtx)
	service2 := NewSeasonSettlementService(svcCtx)
	service1.sendRealtime = func(seasonRolloverRealtimeNotice) error { return nil }
	service2.sendRealtime = func(seasonRolloverRealtimeNotice) error { return nil }
	workers := []*SeasonRolloverWorker{
		newSeasonRolloverWorkerWithDeps(svcCtx, service1, func() time.Time { return now }, time.Minute),
		newSeasonRolloverWorkerWithDeps(svcCtx, service2, func() time.Time { return now }, time.Minute),
	}

	var wg sync.WaitGroup
	errs := make(chan error, len(workers))
	for _, worker := range workers {
		wg.Add(1)
		go func(worker *SeasonRolloverWorker) {
			defer wg.Done()
			errs <- worker.RunOnce(context.Background())
		}(worker)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent worker failed: %v", err)
		}
	}

	assertSeasonStatus(t, svcCtx, season.Id, 2)
	assertSeasonSettlementRecords(t, svcCtx, season.Id)
	assertSeasonSettlementTitles(t, svcCtx, season.Id, 3)
	assertSeasonSettlementSnapshots(t, svcCtx, season.Id, 3)
	assertSeasonRolloverNotifications(t, svcCtx, season, nil, 3)
}
