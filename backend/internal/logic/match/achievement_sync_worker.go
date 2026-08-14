package match

import (
	"context"
	"time"

	"chasing_points/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	achievementSyncWorkerBatchSize = 100
	achievementSyncWorkerInterval  = time.Minute
)

type AchievementSyncWorker struct {
	svcCtx *svc.ServiceContext
}

func NewAchievementSyncWorker(svcCtx *svc.ServiceContext) *AchievementSyncWorker {
	return &AchievementSyncWorker{svcCtx: svcCtx}
}

func (w *AchievementSyncWorker) Start(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	ticker := time.NewTicker(achievementSyncWorkerInterval)
	defer ticker.Stop()
	for {
		if err := w.RunOnce(ctx); err != nil {
			logx.Errorf("对局成就同步任务失败: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *AchievementSyncWorker) RunOnce(ctx context.Context) error {
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
	matches, err := w.svcCtx.MatchModel.ListCompletedPendingAchievementSync(achievementSyncWorkerBatchSize)
	if err != nil {
		return err
	}
	logic := NewFinishMatchLogic(ctx, w.svcCtx)
	for index := range matches {
		if ctx != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}
		if err := logic.syncAchievementProgressForCompletedMatchWithError(&matches[index]); err != nil {
			return err
		}
	}
	return nil
}
