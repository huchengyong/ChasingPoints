package logic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"chasing_points/internal/model"
	seasonx "chasing_points/internal/season"
	"chasing_points/internal/svc"

	"gorm.io/gorm"
)

type SeasonLifecycleResult struct {
	State          string
	Season         *model.Season
	Plan           seasonx.Plan
	Problem        string
	Created        int
	CreatedWindows []seasonx.Window
}

type SeasonLifecycleService struct {
	svcCtx *svc.ServiceContext
}

func NewSeasonLifecycleService(svcCtx *svc.ServiceContext) *SeasonLifecycleService {
	return &SeasonLifecycleService{svcCtx: svcCtx}
}

func (s *SeasonLifecycleService) PlanAt(now time.Time) (*SeasonLifecycleResult, error) {
	if s == nil || s.svcCtx == nil || s.svcCtx.SeasonModel == nil {
		return nil, fmt.Errorf("season lifecycle service is unavailable")
	}
	policy, err := seasonx.NewPolicy(s.svcCtx.Config.SeasonLifecycle)
	if err != nil {
		return &SeasonLifecycleResult{
			State:   seasonx.StateUnavailable,
			Plan:    seasonx.Plan{State: seasonx.StateUnavailable, Windows: []seasonx.Window{}, Missing: []seasonx.Window{}, Conflicts: []string{err.Error()}},
			Problem: err.Error(),
		}, nil
	}
	if !policy.Enabled || !policy.StartedAt(now) {
		return &SeasonLifecycleResult{
			State: seasonx.StateNotStarted,
			Plan:  seasonx.BuildPlan(policy, now, nil),
		}, nil
	}
	seasons, err := s.svcCtx.SeasonModel.ListAll()
	if err != nil {
		return nil, err
	}
	return seasonLifecycleResult(policy, now, seasons), nil
}

func (s *SeasonLifecycleService) EnsureAt(_ context.Context, now time.Time) (*SeasonLifecycleResult, error) {
	if now.IsZero() {
		now = time.Now()
	}
	if s == nil || s.svcCtx == nil || s.svcCtx.DB == nil || s.svcCtx.SeasonModel == nil {
		return nil, fmt.Errorf("season lifecycle service is unavailable")
	}
	policy, err := seasonx.NewPolicy(s.svcCtx.Config.SeasonLifecycle)
	if err != nil {
		return &SeasonLifecycleResult{
			State:   seasonx.StateUnavailable,
			Plan:    seasonx.Plan{State: seasonx.StateUnavailable, Windows: []seasonx.Window{}, Missing: []seasonx.Window{}, Conflicts: []string{err.Error()}},
			Problem: err.Error(),
		}, nil
	}
	if !policy.Enabled || !policy.StartedAt(now) {
		return &SeasonLifecycleResult{State: seasonx.StateNotStarted, Plan: seasonx.BuildPlan(policy, now, nil)}, nil
	}

	var result *SeasonLifecycleResult
	for attempt := 0; attempt < 3; attempt++ {
		result = nil
		createdWindows := []seasonx.Window{}
		err = s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
			seasons, listErr := s.svcCtx.SeasonModel.ListAllWithTx(tx)
			if listErr != nil {
				return listErr
			}
			result = seasonLifecycleResult(policy, now, seasons)
			if len(result.Plan.Conflicts) > 0 {
				return nil
			}
			for _, window := range result.Plan.Missing {
				created, createErr := s.svcCtx.SeasonModel.CreateIfAbsentWithTx(tx, seasonPointer(seasonx.WindowSeason(window)))
				if createErr != nil {
					return createErr
				}
				if created {
					result.Created++
					createdWindows = append(createdWindows, window)
				}
			}

			seasons, listErr = s.svcCtx.SeasonModel.ListAllWithTx(tx)
			if listErr != nil {
				return listErr
			}
			result = seasonLifecycleResult(policy, now, seasons, result.Created)
			result.CreatedWindows = createdWindows
			if len(result.Plan.Conflicts) > 0 || len(result.Plan.Missing) > 0 {
				if result.Problem == "" {
					result.Problem = "season schedule could not converge after creation"
				}
				result.State = seasonx.StateUnavailable
				return nil
			}
			return nil
		})
		if err == nil {
			break
		}
		if !isRetryableSeasonLifecycleError(err) || attempt == 2 {
			return nil, err
		}
		time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
	}
	if result == nil {
		return nil, fmt.Errorf("season lifecycle result missing")
	}
	if result.State != seasonx.StateActive {
		return result, nil
	}

	seasons, err := s.svcCtx.SeasonModel.ListAll()
	if err != nil {
		return nil, err
	}
	final := seasonLifecycleResult(policy, now, seasons, result.Created)
	final.CreatedWindows = append([]seasonx.Window(nil), result.CreatedWindows...)
	return final, nil
}

// ConvergeStatusesAt updates persisted statuses only after callers have
// successfully settled every ended window. Settlement itself keeps each normal
// rollover atomic; this method initializes or repairs already-completed states.
func (s *SeasonLifecycleService) ConvergeStatusesAt(_ context.Context, now time.Time) (*SeasonLifecycleResult, error) {
	if now.IsZero() {
		now = time.Now()
	}
	if s == nil || s.svcCtx == nil || s.svcCtx.DB == nil || s.svcCtx.SeasonModel == nil {
		return nil, fmt.Errorf("season lifecycle service is unavailable")
	}
	policy, err := seasonx.NewPolicy(s.svcCtx.Config.SeasonLifecycle)
	if err != nil {
		return &SeasonLifecycleResult{
			State:   seasonx.StateUnavailable,
			Plan:    seasonx.Plan{State: seasonx.StateUnavailable, Windows: []seasonx.Window{}, Missing: []seasonx.Window{}, Conflicts: []string{err.Error()}},
			Problem: err.Error(),
		}, nil
	}
	if !policy.Enabled || !policy.StartedAt(now) {
		return &SeasonLifecycleResult{State: seasonx.StateNotStarted, Plan: seasonx.BuildPlan(policy, now, nil)}, nil
	}

	var result *SeasonLifecycleResult
	for attempt := 0; attempt < 3; attempt++ {
		result = nil
		err = s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
			seasons, listErr := s.svcCtx.SeasonModel.ListAllWithTx(tx)
			if listErr != nil {
				return listErr
			}
			result = seasonLifecycleResult(policy, now, seasons)
			if result.State != seasonx.StateActive {
				return nil
			}
			for _, window := range result.Plan.Windows {
				for index := range seasons {
					if !seasonx.MatchesWindow(seasons[index], window, policy.Location) {
						continue
					}
					if _, updateErr := s.svcCtx.SeasonModel.UpdateStatusToWithTx(tx, seasons[index].Id, lifecycleStatus(window, now)); updateErr != nil {
						return updateErr
					}
					break
				}
			}
			return nil
		})
		if err == nil {
			break
		}
		if !isRetryableSeasonLifecycleError(err) || attempt == 2 {
			return nil, err
		}
		time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
	}
	if result == nil {
		return nil, fmt.Errorf("season lifecycle result missing")
	}
	if result.State != seasonx.StateActive {
		return result, nil
	}

	seasons, err := s.svcCtx.SeasonModel.ListAll()
	if err != nil {
		return nil, err
	}
	return seasonLifecycleResult(policy, now, seasons), nil
}

func (s *SeasonLifecycleService) Enabled() bool {
	return s != nil && s.svcCtx != nil && s.svcCtx.Config.SeasonLifecycle.Enabled
}

func seasonLifecycleResult(policy seasonx.Policy, now time.Time, seasons []model.Season, created ...int) *SeasonLifecycleResult {
	plan := seasonx.BuildPlan(policy, now, seasons)
	result := &SeasonLifecycleResult{State: plan.State, Plan: plan}
	if len(created) > 0 {
		result.Created = created[0]
	}
	if len(plan.Conflicts) > 0 {
		result.Problem = strings.Join(plan.Conflicts, "; ")
	}
	if plan.Current == nil || plan.State != seasonx.StateActive {
		return result
	}
	for index := range seasons {
		if seasonx.MatchesWindow(seasons[index], *plan.Current, policy.Location) {
			season := seasons[index]
			result.Season = &season
			return result
		}
	}
	result.State = seasonx.StateUnavailable
	result.Problem = "current season window is missing"
	return result
}

func isRetryableSeasonLifecycleError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "database is locked") ||
		strings.Contains(message, "database table is locked") ||
		strings.Contains(message, "deadlock") ||
		strings.Contains(message, "lock wait timeout")
}

func lifecycleStatus(window seasonx.Window, now time.Time) int {
	endExclusive := window.EndDate.AddDate(0, 0, 1)
	if now.Before(window.StartDate) {
		return 0
	}
	if now.Before(endExclusive) {
		return 1
	}
	return 2
}

func seasonPointer(season model.Season) *model.Season {
	return &season
}
