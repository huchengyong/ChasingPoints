package logic

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"chasing_points/internal/config"
	achievementx "chasing_points/internal/logic/achievement"
	"chasing_points/internal/model"
	seasonx "chasing_points/internal/season"
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

func TestSeasonRolloverWorkerLeavesDisabledLifecycleScheduleEmpty(t *testing.T) {
	svcCtx := newSeasonSettlementTestSvc(t)
	worker := newSeasonRolloverWorkerWithDeps(
		svcCtx,
		NewSeasonSettlementService(svcCtx),
		func() time.Time { return time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC) },
		time.Minute,
	)
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("run disabled lifecycle worker: %v", err)
	}
	var count int64
	if err := svcCtx.DB.Model(&model.Season{}).Count(&count).Error; err != nil {
		t.Fatalf("count disabled lifecycle seasons: %v", err)
	}
	if count != 0 {
		t.Fatalf("disabled lifecycle worker must not create seasons, got %d", count)
	}
}

func TestSeasonRolloverWorkerDoesNotFallBackToLegacyWhenLifecycleConfigIsInvalid(t *testing.T) {
	svcCtx := newSeasonSettlementTestSvc(t)
	svcCtx.Config.SeasonLifecycle.Enabled = true
	svcCtx.Config.SeasonLifecycle.AnchorDate = "invalid"
	runner := &fakeSeasonRolloverRunner{}
	worker := newSeasonRolloverWorkerWithDeps(svcCtx, runner, time.Now, time.Minute)

	if err := worker.RunOnce(context.Background()); err == nil {
		t.Fatal("expected invalid lifecycle config to stop worker")
	}
	if runner.settlementCalls != 0 || runner.activationCalls != 0 {
		t.Fatalf("invalid lifecycle config must not run legacy rollover: %+v", runner)
	}
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

func TestSeasonRolloverWorkerEnsuresAndCatchesUpContinuousSchedule(t *testing.T) {
	svcCtx := newSeasonSettlementTestSvc(t)
	svcCtx.Config.SeasonLifecycle.Enabled = true
	svcCtx.Config.SeasonLifecycle.AnchorDate = "2026-07-01"
	svcCtx.Config.SeasonLifecycle.InitialNumber = 1
	svcCtx.Config.SeasonLifecycle.CycleMonths = 1
	svcCtx.Config.SeasonLifecycle.Timezone = "Asia/Shanghai"
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	policy, err := seasonx.NewPolicy(svcCtx.Config.SeasonLifecycle)
	if err != nil {
		t.Fatalf("new lifecycle policy: %v", err)
	}
	for _, at := range []time.Time{
		time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC),
	} {
		window, ok := policy.WindowAt(at)
		if !ok {
			t.Fatal("expected historic lifecycle window")
		}
		season := seasonx.WindowSeason(window)
		if err := svcCtx.SeasonModel.Create(&season); err != nil {
			t.Fatalf("seed historic season %s: %v", season.Name, err)
		}
	}
	runner := &fakeSeasonRolloverRunner{}
	worker := newSeasonRolloverWorkerWithDeps(svcCtx, runner, func() time.Time { return now }, time.Minute)

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("run continuous worker: %v", err)
	}
	if runner.settlementCalls != 3 || runner.activationCalls != 1 {
		t.Fatalf("continuous worker must settle July through September then activate the ready window: settlements=%d activations=%d", runner.settlementCalls, runner.activationCalls)
	}
	seasons, err := svcCtx.SeasonModel.ListAll()
	if err != nil {
		t.Fatalf("list continuous seasons: %v", err)
	}
	if len(seasons) != 5 || seasons[3].Name != "S4" || seasons[3].Status != 0 || seasons[4].Status != 0 {
		t.Fatalf("unexpected continuous schedule: %+v", seasons)
	}
}

func TestSeasonRolloverWorkerPaginatesDueContinuousSeasons(t *testing.T) {
	svcCtx := newSeasonSettlementTestSvc(t)
	svcCtx.Config.SeasonLifecycle = config.SeasonLifecycleConfig{
		Enabled:       true,
		AnchorDate:    "2026-01-01",
		InitialNumber: 1,
		CycleMonths:   1,
		Timezone:      "Asia/Shanghai",
	}
	policy, err := seasonx.NewPolicy(svcCtx.Config.SeasonLifecycle)
	if err != nil {
		t.Fatalf("new lifecycle policy: %v", err)
	}
	for index := 0; index < seasonRolloverBatchSize+5; index++ {
		window, ok := policy.WindowAt(time.Date(2026, time.Month(index+1), 15, 12, 0, 0, 0, time.UTC))
		if !ok {
			t.Fatal("expected due lifecycle window")
		}
		season := seasonx.WindowSeason(window)
		if err := svcCtx.SeasonModel.Create(&season); err != nil {
			t.Fatalf("seed due season %d: %v", index, err)
		}
	}
	runner := &fakeSeasonRolloverRunner{}
	now := time.Date(2028, 4, 1, 12, 0, 0, 0, time.UTC)
	worker := newSeasonRolloverWorkerWithDeps(svcCtx, runner, func() time.Time { return now }, time.Minute)
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("run paged continuous worker: %v", err)
	}
	if runner.settlementCalls != seasonRolloverBatchSize+5 || runner.activationCalls != 1 {
		t.Fatalf("due seasons must be processed in bounded pages then activate once: settlements=%d activations=%d", runner.settlementCalls, runner.activationCalls)
	}
}

func TestSeasonRolloverWorkerDoesNotAdvanceStatusesBeforeFailedContinuousSettlement(t *testing.T) {
	svcCtx := newSeasonSettlementTestSvc(t)
	season, _, now := seedSeasonSettlementScenario(t, svcCtx, false)
	svcCtx.Config.SeasonLifecycle = config.SeasonLifecycleConfig{
		Enabled:       true,
		AnchorDate:    "2026-07-01",
		InitialNumber: 3,
		CycleMonths:   1,
		Timezone:      "Asia/Shanghai",
	}
	if err := svcCtx.DB.Migrator().DropTable(&model.Notification{}); err != nil {
		t.Fatalf("drop notifications: %v", err)
	}
	worker := NewSeasonRolloverWorker(svcCtx)
	worker.now = func() time.Time { return now }

	if err := worker.RunOnce(context.Background()); err == nil {
		t.Fatal("expected continuous settlement failure")
	}
	assertSeasonStatus(t, svcCtx, season.Id, 1)
	seasons, err := svcCtx.SeasonModel.ListAll()
	if err != nil {
		t.Fatalf("list scheduled seasons after failure: %v", err)
	}
	for _, item := range seasons {
		if item.Name == "S4" && item.Status != 0 {
			t.Fatalf("failed settlement must not activate S4: %+v", seasons)
		}
	}

	if err := svcCtx.DB.AutoMigrate(&model.Notification{}); err != nil {
		t.Fatalf("restore notifications: %v", err)
	}
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("retry continuous settlement: %v", err)
	}
	assertSeasonStatus(t, svcCtx, season.Id, 2)
	seasons, err = svcCtx.SeasonModel.ListAll()
	if err != nil {
		t.Fatalf("list scheduled seasons after retry: %v", err)
	}
	for _, item := range seasons {
		if item.Name == "S4" && item.Status != 1 {
			t.Fatalf("successful settlement must atomically activate S4: %+v", item)
		}
	}
}

func TestSeasonRolloverWorkerCountsEventsFromDelayedActivationWindow(t *testing.T) {
	svcCtx := newSeasonSettlementTestSvc(t)
	_, _, now := seedSeasonSettlementScenario(t, svcCtx, false)
	svcCtx.Config.SeasonLifecycle.Enabled = true
	svcCtx.Config.SeasonLifecycle.AnchorDate = "2026-07-01"
	svcCtx.Config.SeasonLifecycle.InitialNumber = 3
	svcCtx.Config.SeasonLifecycle.CycleMonths = 1
	svcCtx.Config.SeasonLifecycle.Timezone = "Asia/Shanghai"
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load timezone: %v", err)
	}
	if _, err := svcCtx.AchievementProgressEventModel.CreateIfAbsent(model.NewAchievementProgressEvent(
		10,
		achievementx.SourceTypeMatch,
		999,
		3,
		achievementx.MetricMatchesTotal,
		5,
		time.Date(2026, 8, 1, 2, 0, 0, 0, location),
	)); err != nil {
		t.Fatalf("seed delayed activation event: %v", err)
	}
	settler := NewSeasonSettlementService(svcCtx)
	settler.sendRealtime = func(seasonRolloverRealtimeNotice) error { return nil }
	worker := newSeasonRolloverWorkerWithDeps(svcCtx, settler, func() time.Time { return now }, time.Minute)
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("run delayed activation worker: %v", err)
	}
	seasons, err := svcCtx.SeasonModel.ListAll()
	if err != nil {
		t.Fatalf("list ensured seasons: %v", err)
	}
	var current *model.Season
	for index := range seasons {
		if seasons[index].Name == "S4" {
			current = &seasons[index]
			break
		}
	}
	if current == nil || current.Status != 1 {
		t.Fatalf("expected active S4 after delayed activation: %+v", seasons)
	}
	progress, err := achievementx.NewSeasonChallengeService(svcCtx).GetProgress(10, current, 3)
	if err != nil {
		t.Fatalf("get delayed activation challenge progress: %v", err)
	}
	if progress[0].Progress != 5 {
		t.Fatalf("events since S4 boundary must not be discarded: %+v", progress)
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
