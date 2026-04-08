package wstsync

import (
	"context"
	"errors"
	"testing"
	"time"

	"chasing_points/internal/config"
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

func TestAutoSyncWorkerBuildSyncParamsUsesSavedCursorWithLookback(t *testing.T) {
	lastSuccess := time.Date(2026, 4, 5, 14, 0, 0, 0, time.UTC)
	store := &fakeAutoSyncStateStore{
		state: &model.WSTSyncJobState{
			JobName:              autoSyncJobName,
			LastSuccessfulSyncAt: &lastSuccess,
		},
	}
	runner := &fakeAutoSyncRunner{}
	worker := NewAutoSyncWorkerWithDeps(store, runner, config.WSTSyncConfig{
		IntervalMinutes: 720,
		LookbackDays:    30,
		LookaheadDays:   7,
		Publish:         true,
	})

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
	worker := NewAutoSyncWorkerWithDeps(store, runner, config.WSTSyncConfig{
		IntervalMinutes: 720,
		LookbackDays:    30,
		LookaheadDays:   7,
		Publish:         true,
	})

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
