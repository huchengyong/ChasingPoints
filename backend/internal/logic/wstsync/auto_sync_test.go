package wstsync

import (
	"context"
	"errors"
	"testing"
	"time"

	"chasing_points/internal/model"
)

type fakeAutoSyncStateStore struct {
	state     *model.WSTSyncJobState
	lastSaved *time.Time
	saveErr   error
	findErr   error
}

func (f *fakeAutoSyncStateStore) FindByJobName(jobName string) (*model.WSTSyncJobState, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	return f.state, nil
}

func (f *fakeAutoSyncStateStore) UpsertLastSuccessfulSyncAt(jobName string, syncedAt time.Time) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	value := syncedAt.UTC()
	f.lastSaved = &value
	return nil
}

type fakeAutoSyncRunner struct {
	params []SyncParams
	runErr error
}

func (f *fakeAutoSyncRunner) Sync(ctx context.Context, params SyncParams) (*SyncSummary, error) {
	f.params = append(f.params, params)
	if f.runErr != nil {
		return nil, f.runErr
	}
	return &SyncSummary{}, nil
}

type fakeHotSyncChecker struct {
	shouldRun bool
	err       error
}

func (f *fakeHotSyncChecker) check(ctx context.Context, now time.Time) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.shouldRun, nil
}

func TestAutoSyncWorkerBuildSyncParamsUsesSavedCursorWithLookback(t *testing.T) {
	lastSuccess := time.Date(2026, 4, 5, 14, 0, 0, 0, time.UTC)
	store := &fakeAutoSyncStateStore{
		state: &model.WSTSyncJobState{
			JobName:              autoSyncJobName,
			LastSuccessfulSyncAt: &lastSuccess,
		},
	}
	runner := &fakeAutoSyncRunner{}
	worker := NewAutoSyncWorkerWithDeps(autoSyncJobName, store, runner, 720, 30, 7, true, true, nil)

	now := time.Date(2026, 4, 8, 9, 30, 0, 0, time.UTC)
	worker.now = func() time.Time { return now }

	params, err := worker.buildSyncParams(now)
	if err != nil {
		t.Fatalf("build sync params: %v", err)
	}

	wantFrom := time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC)
	wantTo := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	if params.From == nil || !params.From.Equal(wantFrom) {
		t.Fatalf("expected from %v, got %#v", wantFrom, params.From)
	}
	if params.To == nil || !params.To.Equal(wantTo) {
		t.Fatalf("expected to %v, got %#v", wantTo, params.To)
	}
}

func TestAutoSyncWorkerRunOncePersistsCursorOnlyAfterSuccessfulSync(t *testing.T) {
	store := &fakeAutoSyncStateStore{}
	runner := &fakeAutoSyncRunner{}
	worker := NewAutoSyncWorkerWithDeps(autoSyncJobName, store, runner, 720, 30, 7, true, true, nil)

	runAt := time.Date(2026, 4, 8, 11, 0, 0, 0, time.UTC)
	worker.now = func() time.Time { return runAt }

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if len(runner.params) != 1 {
		t.Fatalf("expected runner called once, got %d", len(runner.params))
	}
	if store.lastSaved == nil || !store.lastSaved.Equal(runAt) {
		t.Fatalf("expected saved cursor %v, got %#v", runAt, store.lastSaved)
	}

	store.lastSaved = nil
	runner.runErr = errors.New("boom")
	if err := worker.RunOnce(context.Background()); err == nil {
		t.Fatal("expected sync error")
	}
	if store.lastSaved != nil {
		t.Fatalf("expected cursor unchanged on failure, got %#v", store.lastSaved)
	}
}

func TestHotAutoSyncWorkerUsesRollingWindowWithoutCursorPersistence(t *testing.T) {
	store := &fakeAutoSyncStateStore{
		state: &model.WSTSyncJobState{
			JobName:              hotAutoSyncJobName,
			LastSuccessfulSyncAt: func() *time.Time { value := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC); return &value }(),
		},
	}
	runner := &fakeAutoSyncRunner{}
	checker := &fakeHotSyncChecker{shouldRun: true}
	worker := NewAutoSyncWorkerWithDeps(hotAutoSyncJobName, store, runner, 15, 2, 7, true, false, checker.check)

	runAt := time.Date(2026, 4, 8, 11, 0, 0, 0, time.UTC)
	worker.now = func() time.Time { return runAt }

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if len(runner.params) != 1 {
		t.Fatalf("expected runner called once, got %d", len(runner.params))
	}
	wantFrom := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)
	wantTo := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	if runner.params[0].From == nil || !runner.params[0].From.Equal(wantFrom) {
		t.Fatalf("expected hot sync from %v, got %#v", wantFrom, runner.params[0].From)
	}
	if runner.params[0].To == nil || !runner.params[0].To.Equal(wantTo) {
		t.Fatalf("expected hot sync to %v, got %#v", wantTo, runner.params[0].To)
	}
	if store.lastSaved != nil {
		t.Fatalf("expected hot sync to skip cursor persistence, got %#v", store.lastSaved)
	}
}

func TestHotAutoSyncWorkerSkipsWhenNoHotTargets(t *testing.T) {
	store := &fakeAutoSyncStateStore{}
	runner := &fakeAutoSyncRunner{}
	checker := &fakeHotSyncChecker{shouldRun: false}
	worker := NewAutoSyncWorkerWithDeps(hotAutoSyncJobName, store, runner, 15, 2, 7, true, false, checker.check)

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if len(runner.params) != 0 {
		t.Fatalf("expected no sync call when no hot targets, got %d", len(runner.params))
	}
}
