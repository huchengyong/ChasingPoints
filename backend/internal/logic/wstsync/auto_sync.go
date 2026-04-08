package wstsync

import (
	"context"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

const autoSyncJobName = "wst-auto-sync"

type AutoSyncStateStore interface {
	FindByJobName(jobName string) (*model.WSTSyncJobState, error)
	UpsertLastSuccessfulSyncAt(jobName string, syncedAt time.Time) error
}

type AutoSyncRunner interface {
	Sync(ctx context.Context, params SyncParams) (*SyncSummary, error)
}

type AutoSyncWorker struct {
	logger        logx.Logger
	stateStore    AutoSyncStateStore
	runner        AutoSyncRunner
	interval      time.Duration
	lookbackDays  int
	lookaheadDays int
	publish       bool
	now           func() time.Time
}

func NewAutoSyncWorker(svcCtx *svc.ServiceContext, cfg config.WSTSyncConfig) *AutoSyncWorker {
	client := NewClient(DefaultSeasonsURL, DefaultTournamentsURL, DefaultMatchesURL)
	return NewAutoSyncWorkerWithDeps(
		model.NewWSTSyncJobStateModel(svcCtx.DB),
		NewService(svcCtx, client),
		cfg,
	)
}

func NewAutoSyncWorkerWithDeps(stateStore AutoSyncStateStore, runner AutoSyncRunner, cfg config.WSTSyncConfig) *AutoSyncWorker {
	interval := time.Duration(cfg.IntervalMinutes) * time.Minute
	if interval <= 0 {
		interval = 12 * time.Hour
	}
	lookbackDays := cfg.LookbackDays
	if lookbackDays <= 0 {
		lookbackDays = 30
	}
	lookaheadDays := cfg.LookaheadDays
	if lookaheadDays < 0 {
		lookaheadDays = 0
	}

	return &AutoSyncWorker{
		logger:        logx.WithContext(context.Background()),
		stateStore:    stateStore,
		runner:        runner,
		interval:      interval,
		lookbackDays:  lookbackDays,
		lookaheadDays: lookaheadDays,
		publish:       cfg.Publish,
		now:           time.Now,
	}
}

func (w *AutoSyncWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		if err := w.RunOnce(ctx); err != nil {
			w.logger.Errorf("wst auto sync run once failed: %v", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *AutoSyncWorker) RunOnce(ctx context.Context) error {
	runAt := w.now().UTC()
	params, err := w.buildSyncParams(runAt)
	if err != nil {
		return err
	}

	if _, err := w.runner.Sync(ctx, params); err != nil {
		return err
	}

	return w.stateStore.UpsertLastSuccessfulSyncAt(autoSyncJobName, runAt)
}

func (w *AutoSyncWorker) buildSyncParams(now time.Time) (SyncParams, error) {
	from := normalizeAutoSyncDay(now.AddDate(0, 0, -w.lookbackDays))
	to := normalizeAutoSyncDay(now.AddDate(0, 0, w.lookaheadDays))

	state, err := w.stateStore.FindByJobName(autoSyncJobName)
	if err != nil {
		return SyncParams{}, err
	}
	if state != nil && state.LastSuccessfulSyncAt != nil && !state.LastSuccessfulSyncAt.IsZero() {
		from = normalizeAutoSyncDay(state.LastSuccessfulSyncAt.AddDate(0, 0, -w.lookbackDays))
	}

	return SyncParams{
		Mode:              SyncModeRange,
		From:              &from,
		To:                &to,
		Publish:           w.publish,
		DryRun:            false,
		IncludeQualifiers: true,
		GameType:          1,
	}, nil
}

func normalizeAutoSyncDay(t time.Time) time.Time {
	utc := t.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}
