package match

import (
	"context"
	"time"

	"chasing_points/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	finishRequestExpiryWorkerBatchSize = 100
	finishRequestExpiryWorkerInterval  = time.Minute
)

type FinishRequestExpiryWorker struct {
	svcCtx *svc.ServiceContext
}

func NewFinishRequestExpiryWorker(svcCtx *svc.ServiceContext) *FinishRequestExpiryWorker {
	return &FinishRequestExpiryWorker{svcCtx: svcCtx}
}

func (w *FinishRequestExpiryWorker) Start(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	ticker := time.NewTicker(finishRequestExpiryWorkerInterval)
	defer ticker.Stop()
	for {
		if err := w.RunOnce(ctx); err != nil {
			logx.Errorf("结束请求过期任务失败: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *FinishRequestExpiryWorker) RunOnce(ctx context.Context) error {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	if w == nil || w.svcCtx == nil || w.svcCtx.MatchModel == nil {
		return nil
	}
	ids, err := w.svcCtx.MatchModel.ListStaleFinishRequestIDs(finishRequestExpiryWorkerBatchSize, time.Now())
	if err != nil {
		return err
	}
	for _, matchID := range ids {
		if ctx != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}
		match, expired, _, err := w.svcCtx.MatchModel.ExpireStaleFinishRequest(matchID)
		if err != nil {
			return err
		}
		if expired && match != nil {
			broadcastFinishActionState(w.svcCtx, match, "match_finish_expired")
		}
	}
	return nil
}
