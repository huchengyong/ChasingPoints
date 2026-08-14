package season

import (
	"time"

	"chasing_points/internal/model"
	seasonx "chasing_points/internal/season"
	"chasing_points/internal/svc"
)

// ResolveCurrentSeasonLifecycle resolves only the deterministic current window on
// request paths. Full schedule reconciliation belongs to lifecycle workers.
func ResolveCurrentSeasonLifecycle(svcCtx *svc.ServiceContext, now time.Time) (*model.Season, string, error) {
	if svcCtx == nil || svcCtx.SeasonModel == nil {
		return nil, seasonx.StateUnavailable, nil
	}
	return seasonx.ResolveCurrentLifecycle(svcCtx.Config.SeasonLifecycle, now, svcCtx.SeasonModel)
}
