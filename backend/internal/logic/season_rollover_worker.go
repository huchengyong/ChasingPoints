package logic

import (
	"context"
	"fmt"
	"time"

	"chasing_points/internal/model"
	seasonx "chasing_points/internal/season"
	"chasing_points/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

const defaultSeasonRolloverInterval = time.Minute

type seasonRolloverRunner interface {
	SettleSeasonAt(ctx context.Context, seasonId int64, now time.Time) (*SeasonSettlementSummary, error)
	ActivateReadySeasonAt(ctx context.Context, now time.Time) (*model.Season, error)
}

type SeasonRolloverWorker struct {
	svcCtx    *svc.ServiceContext
	runner    seasonRolloverRunner
	lifecycle *SeasonLifecycleService
	now       func() time.Time
	interval  time.Duration
}

func NewSeasonRolloverWorker(svcCtx *svc.ServiceContext) *SeasonRolloverWorker {
	return newSeasonRolloverWorkerWithDeps(
		svcCtx,
		NewSeasonSettlementService(svcCtx),
		time.Now,
		defaultSeasonRolloverInterval,
	)
}

func newSeasonRolloverWorkerWithDeps(
	svcCtx *svc.ServiceContext,
	runner seasonRolloverRunner,
	now func() time.Time,
	interval time.Duration,
) *SeasonRolloverWorker {
	if now == nil {
		now = time.Now
	}
	if interval <= 0 {
		interval = defaultSeasonRolloverInterval
	}
	return &SeasonRolloverWorker{
		svcCtx:    svcCtx,
		runner:    runner,
		lifecycle: NewSeasonLifecycleService(svcCtx),
		now:       now,
		interval:  interval,
	}
}

func (w *SeasonRolloverWorker) RunOnce(ctx context.Context) error {
	if w == nil || w.svcCtx == nil || w.svcCtx.SeasonModel == nil || w.svcCtx.SeasonSettlementModel == nil || w.runner == nil {
		return fmt.Errorf("season rollover worker is unavailable")
	}
	now := w.now()
	if w.lifecycle != nil && w.lifecycle.Enabled() {
		result, err := w.lifecycle.EnsureAt(ctx, now)
		if err != nil {
			return err
		}
		if result.State == seasonx.StateUnavailable {
			return fmt.Errorf("season lifecycle unavailable: %s", result.Problem)
		}
		if result.State == seasonx.StateNotStarted {
			return nil
		}
		if err := w.settleContinuousSeasons(ctx, now); err != nil {
			return err
		}
		_, err = w.lifecycle.ConvergeStatusesAt(ctx, now)
		return err
	}
	return w.runLegacyRollover(ctx, now)
}

func (w *SeasonRolloverWorker) runLegacyRollover(ctx context.Context, now time.Time) error {
	activeSeasons, err := w.svcCtx.SeasonModel.ListActive()
	if err != nil {
		return err
	}
	for _, season := range activeSeasons {
		_, endExclusive, boundsErr := seasonx.BoundsForConfig(w.svcCtx.Config.SeasonLifecycle, &season)
		if boundsErr != nil {
			return boundsErr
		}
		if now.Before(endExclusive) {
			continue
		}
		settlement, err := w.svcCtx.SeasonSettlementModel.FindBySeasonId(season.Id)
		if err != nil {
			return err
		}
		if settlement != nil && settlement.Status == model.SeasonSettlementStatusCompleted {
			continue
		}
		if _, err := w.runner.SettleSeasonAt(ctx, season.Id, now); err != nil {
			return err
		}
	}
	_, err = w.runner.ActivateReadySeasonAt(ctx, now)
	return err
}

func (w *SeasonRolloverWorker) settleContinuousSeasons(ctx context.Context, now time.Time) error {
	policy, err := seasonx.NewPolicy(w.svcCtx.Config.SeasonLifecycle)
	if err != nil {
		return err
	}
	seasons, err := w.svcCtx.SeasonModel.ListAll()
	if err != nil {
		return err
	}
	for _, item := range seasons {
		startAt, endExclusive := seasonx.Bounds(&item, policy.Location)
		if startAt.Before(policy.Anchor) || now.Before(endExclusive) {
			continue
		}
		settlement, err := w.svcCtx.SeasonSettlementModel.FindBySeasonId(item.Id)
		if err != nil {
			return err
		}
		if settlement != nil && settlement.Status == model.SeasonSettlementStatusCompleted {
			continue
		}
		if _, err := w.runner.SettleSeasonAt(ctx, item.Id, now); err != nil {
			return err
		}
	}
	return nil
}

func (w *SeasonRolloverWorker) Start(ctx context.Context) {
	if err := w.RunOnce(ctx); err != nil {
		logx.WithContext(ctx).Errorf("赛季换季任务执行失败: %v", err)
	}
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.RunOnce(ctx); err != nil {
				logx.WithContext(ctx).Errorf("赛季换季任务执行失败: %v", err)
			}
		}
	}
}
