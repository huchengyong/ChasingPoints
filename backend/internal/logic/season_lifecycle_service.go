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

// EnsureAt is the normal lifecycle-worker fast path. It only verifies and
// creates the deterministic current and next windows; repair owns history-wide
// schedule reconciliation.
func (s *SeasonLifecycleService) EnsureAt(ctx context.Context, now time.Time) (*SeasonLifecycleResult, error) {
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
			current, resolveErr := expectedLifecycleResult(policy, now, s.svcCtx.SeasonModel, tx)
			if resolveErr != nil {
				return resolveErr
			}
			result = current
			if len(result.Plan.Conflicts) > 0 || len(result.Plan.Missing) == 0 {
				return nil
			}
			for _, window := range result.Plan.Missing {
				created, createErr := s.svcCtx.SeasonModel.CreateIfAbsentWithTx(tx, seasonPointer(seasonx.WindowSeason(window)))
				if createErr != nil {
					return createErr
				}
				if created {
					createdWindows = append(createdWindows, window)
				}
			}
			result, resolveErr = expectedLifecycleResult(policy, now, s.svcCtx.SeasonModel, tx)
			if resolveErr != nil {
				return resolveErr
			}
			result.Created = len(createdWindows)
			result.CreatedWindows = createdWindows
			if len(result.Plan.Conflicts) > 0 || len(result.Plan.Missing) > 0 {
				if result.Problem == "" {
					result.Problem = "season schedule could not converge after creation"
				}
				result.State = seasonx.StateUnavailable
				result.Plan.State = seasonx.StateUnavailable
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
	if result.Created > 0 {
		invalidateSeasonInfoCache(ctx, s.svcCtx)
	}
	return result, nil
}

func (s *SeasonLifecycleService) Enabled() bool {
	return s != nil && s.svcCtx != nil && s.svcCtx.Config.SeasonLifecycle.Enabled
}

func expectedLifecycleResult(policy seasonx.Policy, now time.Time, seasonModel *model.SeasonModel, tx *gorm.DB) (*SeasonLifecycleResult, error) {
	current, started := policy.WindowAt(now)
	if !started {
		return &SeasonLifecycleResult{State: seasonx.StateNotStarted, Plan: seasonx.BuildPlan(policy, now, nil)}, nil
	}
	nextStart := current.StartDate.AddDate(0, policy.CycleMonths, 0)
	next := seasonx.Window{
		Number: current.Number + 1, Name: fmt.Sprintf("S%d", current.Number+1), StartDate: nextStart,
		EndDate: nextStart.AddDate(0, policy.CycleMonths, 0).AddDate(0, 0, -1),
	}
	windows := []seasonx.Window{current, next}
	result := &SeasonLifecycleResult{
		State: seasonx.StateActive,
		Plan:  seasonx.Plan{State: seasonx.StateActive, Current: &current, Windows: windows, Missing: []seasonx.Window{}, Conflicts: []string{}},
	}
	for index, window := range windows {
		candidates, err := seasonModel.FindByStartDateCandidatesWithTx(tx, window.StartDate, false)
		if err != nil {
			return nil, err
		}
		if len(candidates) == 0 {
			result.Plan.Missing = append(result.Plan.Missing, window)
			continue
		}
		if len(candidates) != 1 || !seasonx.MatchesWindow(candidates[0], window, policy.Location) {
			name := candidates[0].Name
			if len(candidates) > 1 {
				name = "multiple seasons"
			}
			result.Plan.Conflicts = append(result.Plan.Conflicts, fmt.Sprintf("season %q conflicts with deterministic window %s", name, window.Name))
			continue
		}
		if index == 0 {
			season := candidates[0]
			result.Season = &season
		}
	}
	if len(result.Plan.Conflicts) > 0 || result.Season == nil {
		result.State = seasonx.StateUnavailable
		result.Plan.State = seasonx.StateUnavailable
		result.Problem = strings.Join(result.Plan.Conflicts, "; ")
		if result.Problem == "" {
			result.Problem = "current season window is missing"
		}
	}
	return result, nil
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
