package rank

import (
	"context"
	"time"

	logicx "chasing_points/internal/logic"
	publiclogic "chasing_points/internal/logic/public"
	"chasing_points/internal/model"
	"chasing_points/internal/ranking"
	"chasing_points/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

const defaultRatingRulesInterval = time.Minute

// RatingRulesWorker 在排位规则 V2 生效后把仍是旧规则的段位行切换到 V2，保证无人对局的用户也在一分钟内看到新分。
// 稳定状态只执行一次 idx_user_ranking_rating_version 上的 LIMIT 1 检查。
type RatingRulesWorker struct {
	svcCtx   *svc.ServiceContext
	now      func() time.Time
	interval time.Duration
}

func NewRatingRulesWorker(svcCtx *svc.ServiceContext) *RatingRulesWorker {
	return &RatingRulesWorker{svcCtx: svcCtx, now: time.Now, interval: defaultRatingRulesInterval}
}

// RunOnce 返回本次切换的段位行数。
func (w *RatingRulesWorker) RunOnce(ctx context.Context) (int, error) {
	if w == nil || w.svcCtx == nil || w.svcCtx.RankingModel == nil {
		return 0, nil
	}
	now := w.now()
	if !ranking.RulesFor(w.svcCtx.Config).V2ActiveAt(now) {
		return 0, nil
	}
	pending, err := w.svcCtx.RankingModel.HasRankingsBeforeRatingVersion(nil, model.RatingVersionV2)
	if err != nil || !pending {
		return 0, err
	}
	switched, err := logicx.SwitchPendingRankingsToRatingV2(w.svcCtx, now)
	if switched > 0 && w.svcCtx.Redis != nil {
		for _, gameType := range model.SupportedRankingGameTypes() {
			if bumpErr := publiclogic.BumpLeaderboardCacheVersion(ctx, w.svcCtx, gameType); bumpErr != nil {
				logx.WithContext(ctx).Errorf("排位规则切换后失效排行榜缓存失败: gameType=%d err=%v", gameType, bumpErr)
			}
		}
	}
	return switched, err
}

func (w *RatingRulesWorker) Start(ctx context.Context) {
	if err := ranking.RulesFor(w.svcCtx.Config).Err(); err != nil {
		logx.WithContext(ctx).Errorf("排位规则 V2 配置无效，继续使用旧规则: %v", err)
	}
	w.runAndLog(ctx)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runAndLog(ctx)
		}
	}
}

func (w *RatingRulesWorker) runAndLog(ctx context.Context) {
	switched, err := w.RunOnce(ctx)
	if err != nil {
		logx.WithContext(ctx).Errorf("排位规则 V2 切换任务执行失败: switched=%d err=%v", switched, err)
		return
	}
	if switched > 0 {
		logx.WithContext(ctx).Infof("排位规则 V2 切换段位行: %d", switched)
	}
}
