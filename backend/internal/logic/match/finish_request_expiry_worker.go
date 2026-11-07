package match

import (
	"context"
	"time"

	"chasing_points/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	finishRequestExpiryWorkerBatchSize = 100
	// 确认期只有30秒，按秒级扫描让期满后的结算延迟控制在数秒内；沿用 idx_matches_finish_request_expiry。
	finishRequestExpiryWorkerInterval = 2 * time.Second
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
	if w == nil || w.svcCtx == nil || w.svcCtx.MatchModel == nil || w.svcCtx.DB == nil {
		return nil
	}
	ids, err := w.svcCtx.MatchModel.ListStaleFinishRequestIDs(finishRequestExpiryWorkerBatchSize, time.Now())
	if err != nil {
		return err
	}
	completed := 0
	var oldestDeadline time.Time
	for _, matchID := range ids {
		if ctx != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}
		// 单场失败只记录并继续，避免一条异常数据挡住整批到期请求。
		completion, err := completeExpiredFinishRequest(ctx, w.svcCtx, matchID, time.Now())
		if err != nil {
			logx.Errorf("确认期满结算失败: matchId=%d err=%v", matchID, err)
			continue
		}
		if completion != nil {
			completed++
			if oldestDeadline.IsZero() || completion.deadline.Before(oldestDeadline) {
				oldestDeadline = completion.deadline
			}
		}
	}
	if len(ids) > 0 {
		logx.Infof("确认期满结算: 到期 %d 场，完成 %d 场，最早截止 %s", len(ids), completed, oldestDeadline.Format(time.RFC3339))
		if len(ids) == finishRequestExpiryWorkerBatchSize {
			logx.Errorf("确认期满结算达到单批上限 %d，可能仍有积压", finishRequestExpiryWorkerBatchSize)
		}
	}
	return nil
}
