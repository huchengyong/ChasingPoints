package season

import (
	"time"

	"chasing_points/internal/model"
	seasonx "chasing_points/internal/season"
	"chasing_points/internal/svc"
)

func resolveCurrentSeasonLifecycle(svcCtx *svc.ServiceContext, now time.Time) (*model.Season, string, error) {
	if svcCtx == nil || svcCtx.SeasonModel == nil {
		return nil, seasonx.StateUnavailable, nil
	}
	policy, err := seasonx.NewPolicy(svcCtx.Config.SeasonLifecycle)
	if err != nil {
		return nil, seasonx.StateUnavailable, nil
	}
	if !policy.Enabled {
		current, findErr := svcCtx.SeasonModel.FindCurrent()
		if findErr != nil {
			return nil, seasonx.StateUnavailable, findErr
		}
		if current != nil {
			return current, seasonx.StateActive, nil
		}
		return nil, seasonx.StateNotStarted, nil
	}
	if !policy.StartedAt(now) {
		return nil, seasonx.StateNotStarted, nil
	}
	seasons, err := svcCtx.SeasonModel.ListAll()
	if err != nil {
		return nil, seasonx.StateUnavailable, err
	}
	resolved := seasonx.Resolve(policy, now, seasons)
	if resolved.Season != nil && resolved.State == seasonx.StateActive {
		current := *resolved.Season
		current.Status = 1
		return &current, resolved.State, nil
	}
	return resolved.Season, resolved.State, nil
}
