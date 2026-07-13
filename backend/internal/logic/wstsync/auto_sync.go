package wstsync

import (
	"context"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	autoSyncJobName    = "wst-auto-sync"
	hotAutoSyncJobName = "wst-hot-sync"
)

type AutoSyncStateStore interface {
	FindByJobName(jobName string) (*model.WSTSyncJobState, error)
	UpsertLastSuccessfulSyncAt(jobName string, syncedAt time.Time) error
}

type AutoSyncRunner interface {
	Sync(ctx context.Context, params SyncParams) (*SyncSummary, error)
}

type AutoSyncWorker struct {
	logger        logx.Logger
	jobName       string
	stateStore    AutoSyncStateStore
	runner        AutoSyncRunner
	interval      time.Duration
	lookbackDays  int
	lookaheadDays int
	useCursor     bool
	publish       bool
	shouldRun     func(ctx context.Context, now time.Time) (bool, error)
	now           func() time.Time
}

func NewAutoSyncWorker(svcCtx *svc.ServiceContext, cfg config.WSTSyncConfig) *AutoSyncWorker {
	client := NewClient(DefaultSeasonsURL, DefaultTournamentsURL, DefaultMatchesURL)
	return NewAutoSyncWorkerWithDeps(
		autoSyncJobName,
		model.NewWSTSyncJobStateModel(svcCtx.DB),
		NewService(svcCtx, client),
		cfg.IntervalMinutes,
		cfg.LookbackDays,
		cfg.LookaheadDays,
		cfg.Publish,
		true,
		nil,
	)
}

func NewHotAutoSyncWorker(svcCtx *svc.ServiceContext, cfg config.WSTSyncConfig) *AutoSyncWorker {
	client := NewClient(DefaultSeasonsURL, DefaultTournamentsURL, DefaultMatchesURL)
	checker := svcCtx.TournamentModel
	return NewAutoSyncWorkerWithDeps(
		hotAutoSyncJobName,
		model.NewWSTSyncJobStateModel(svcCtx.DB),
		NewService(svcCtx, client),
		cfg.HotIntervalMinutes,
		cfg.HotLookbackDays,
		cfg.HotLookaheadDays,
		cfg.Publish,
		false,
		func(ctx context.Context, now time.Time) (bool, error) {
			return checker.HasOfficialTournamentsNeedingHotSync(now, cfg.HotLookbackDays, cfg.HotLookaheadDays)
		},
	)
}

func NewAutoSyncWorkerWithDeps(
	jobName string,
	stateStore AutoSyncStateStore,
	runner AutoSyncRunner,
	intervalMinutes int,
	lookbackDays int,
	lookaheadDays int,
	publish bool,
	useCursor bool,
	shouldRun func(ctx context.Context, now time.Time) (bool, error),
) *AutoSyncWorker {
	interval := time.Duration(intervalMinutes) * time.Minute
	if interval <= 0 {
		interval = 12 * time.Hour
	}
	if lookbackDays <= 0 {
		lookbackDays = 30
	}
	if lookaheadDays < 0 {
		lookaheadDays = 0
	}

	return &AutoSyncWorker{
		logger:        logx.WithContext(context.Background()),
		jobName:       jobName,
		stateStore:    stateStore,
		runner:        runner,
		interval:      interval,
		lookbackDays:  lookbackDays,
		lookaheadDays: lookaheadDays,
		useCursor:     useCursor,
		publish:       publish,
		shouldRun:     shouldRun,
		now:           time.Now,
	}
}

func (w *AutoSyncWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		if err := w.RunOnce(ctx); err != nil {
			w.logger.Errorf("%s run once failed: %v", w.jobName, err)
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

	if w.shouldRun != nil {
		shouldRun, err := w.shouldRun(ctx, runAt)
		if err != nil {
			return err
		}
		if !shouldRun {
			return nil
		}
	}

	params, err := w.buildSyncParams(runAt)
	if err != nil {
		return err
	}

	if _, err := w.runner.Sync(ctx, params); err != nil {
		return err
	}

	if !w.useCursor || w.stateStore == nil {
		return nil
	}
	return w.stateStore.UpsertLastSuccessfulSyncAt(w.jobName, runAt)
}

func (w *AutoSyncWorker) buildSyncParams(now time.Time) (SyncParams, error) {
	from := normalizeAutoSyncDay(now.AddDate(0, 0, -w.lookbackDays))
	to := normalizeAutoSyncDay(now.AddDate(0, 0, w.lookaheadDays))

	if w.useCursor && w.stateStore != nil {
		state, err := w.stateStore.FindByJobName(w.jobName)
		if err != nil {
			return SyncParams{}, err
		}
		if state != nil && state.LastSuccessfulSyncAt != nil && !state.LastSuccessfulSyncAt.IsZero() {
			from = normalizeAutoSyncDay(state.LastSuccessfulSyncAt.AddDate(0, 0, -w.lookbackDays))
		}
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
