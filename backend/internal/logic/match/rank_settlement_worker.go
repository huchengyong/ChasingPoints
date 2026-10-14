package match

import (
	"context"
	"time"

	"chasing_points/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	rankSettlementWorkerBatchSize = 100
	rankSettlementWorkerInterval  = time.Minute
)

// RankSettlementWorker 在异议期结束后完成约球排位的延后结算，不依赖任何用户在线或打开页面。
type RankSettlementWorker struct {
	svcCtx *svc.ServiceContext
}

func NewRankSettlementWorker(svcCtx *svc.ServiceContext) *RankSettlementWorker {
	return &RankSettlementWorker{svcCtx: svcCtx}
}

func (w *RankSettlementWorker) Start(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	ticker := time.NewTicker(rankSettlementWorkerInterval)
	defer ticker.Stop()
	for {
		if err := w.RunOnce(ctx); err != nil {
			logx.Errorf("排位延后结算任务失败: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// RunOnce 处理一批已到期的待生效比赛。单场失败只记录并继续，下个周期会重试。
func (w *RankSettlementWorker) RunOnce(ctx context.Context) error {
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
	if ctx == nil {
		ctx = context.Background()
	}
	now := time.Now()
	ids, err := w.svcCtx.MatchModel.ListDueRankSettlementIDs(rankSettlementWorkerBatchSize, now)
	if err != nil {
		return err
	}
	if len(ids) == rankSettlementWorkerBatchSize {
		// 满批说明可能有积压：记录规模与最早到期时刻，便于观测 worker 是否跟得上。
		if total, earliest, backlogErr := w.svcCtx.MatchModel.RankSettlementBacklog(now); backlogErr == nil {
			logx.Infof("排位延后结算积压: due=%d earliest=%v", total, earliest)
		}
	}
	for _, matchID := range ids {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		match, settlement, kind, err := settleDueRankMatch(ctx, w.svcCtx, matchID, now)
		if err != nil {
			logx.Errorf("排位延后结算失败: matchId=%d err=%v", matchID, err)
			continue
		}
		logic := NewFinishMatchLogic(ctx, w.svcCtx)
		switch kind {
		case rankSettlementApplied:
			logic.finalizeRankSettled(match, settlement)
		case rankSettlementVoidedByMissingUser:
			logic.finalizeRankVoided(match)
		}
	}
	return nil
}
