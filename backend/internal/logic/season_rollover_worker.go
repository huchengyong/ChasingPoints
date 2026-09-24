package logic

import (
	"context"
	"fmt"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

const defaultSeasonRolloverInterval = time.Minute

type seasonRolloverRunner interface {
	SettleSeasonAt(ctx context.Context, seasonId int64, now time.Time) (*SeasonSettlementSummary, error)
	ActivateReadySeasonAt(ctx context.Context, now time.Time) (*model.Season, error)
}

type SeasonRolloverWorker struct {
	svcCtx   *svc.ServiceContext
	runner   seasonRolloverRunner
	now      func() time.Time
	interval time.Duration
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
		svcCtx:   svcCtx,
		runner:   runner,
		now:      now,
		interval: interval,
	}
}

func (w *SeasonRolloverWorker) RunOnce(ctx context.Context) error {
	if w == nil || w.svcCtx == nil || w.svcCtx.SeasonModel == nil || w.svcCtx.SeasonSettlementModel == nil || w.runner == nil {
		return fmt.Errorf("season rollover worker is unavailable")
	}
	now := w.now()
	activeSeasons, err := w.svcCtx.SeasonModel.ListActive()
	if err != nil {
		return err
	}
	for _, season := range activeSeasons {
		if now.Before(seasonSettlementEndExclusive(season.EndDate)) {
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
