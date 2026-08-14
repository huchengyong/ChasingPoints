package logic

import (
	"context"
	"time"

	"chasing_points/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	challengeExpiryWorkerBatchSize = 100
	challengeExpiryWorkerInterval  = time.Minute
)

type ChallengeExpiryWorker struct {
	svcCtx *svc.ServiceContext
}

func NewChallengeExpiryWorker(svcCtx *svc.ServiceContext) *ChallengeExpiryWorker {
	return &ChallengeExpiryWorker{svcCtx: svcCtx}
}

func (w *ChallengeExpiryWorker) Start(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	ticker := time.NewTicker(challengeExpiryWorkerInterval)
	defer ticker.Stop()
	for {
		if err := w.RunOnce(ctx); err != nil {
			logx.Errorf("挑战过期任务失败: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *ChallengeExpiryWorker) RunOnce(ctx context.Context) error {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	if w == nil || w.svcCtx == nil || w.svcCtx.ChallengeModel == nil {
		return nil
	}
	now := time.Now()
	challenges, err := w.svcCtx.ChallengeModel.ListExpiredPending(challengeExpiryWorkerBatchSize, now)
	if err != nil {
		return err
	}
	for _, challenge := range challenges {
		if ctx != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}
		if _, err := w.svcCtx.ChallengeModel.MarkExpiredIfPending(challenge.Id, now); err != nil {
			return err
		}
	}
	return nil
}
