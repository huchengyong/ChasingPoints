package season

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
)

const (
	StateNotStarted  = "not_started"
	StateActive      = "active"
	StateUnavailable = "unavailable"

	defaultTimezone    = "Asia/Shanghai"
	defaultCycleMonths = 1
	defaultSeasonStart = 1
)

// Policy describes the one-time schedule used to derive every continuous season.
type Policy struct {
	Enabled       bool
	Anchor        time.Time
	InitialNumber int
	CycleMonths   int
	Location      *time.Location
}

// Window is a deterministic, date-only season interval.
type Window struct {
	Number    int
	Name      string
	StartDate time.Time
	EndDate   time.Time
}

// Plan describes the expected windows around a point in time and any unsafe
// differences found in persisted season rows.
type Plan struct {
	State     string
	Current   *Window
	Windows   []Window
	Missing   []Window
	Conflicts []string
}

type Resolution struct {
	State   string
	Season  *model.Season
	Plan    Plan
	Problem string
}

func NewPolicy(cfg config.SeasonLifecycleConfig) (Policy, error) {
	locationName := strings.TrimSpace(cfg.Timezone)
	if locationName == "" {
		locationName = defaultTimezone
	}
	location, err := time.LoadLocation(locationName)
	if err != nil {
		return Policy{}, fmt.Errorf("load season lifecycle timezone %q: %w", locationName, err)
	}

	initialNumber := cfg.InitialNumber
	if initialNumber == 0 {
		initialNumber = defaultSeasonStart
	}
	if initialNumber < 1 {
		return Policy{}, fmt.Errorf("season lifecycle initial number must be positive")
	}
	cycleMonths := cfg.CycleMonths
	if cycleMonths == 0 {
		cycleMonths = defaultCycleMonths
	}
	if cycleMonths < 1 {
		return Policy{}, fmt.Errorf("season lifecycle cycle months must be positive")
	}

	policy := Policy{
		Enabled:       cfg.Enabled,
		InitialNumber: initialNumber,
		CycleMonths:   cycleMonths,
		Location:      location,
	}
	if !policy.Enabled {
		return policy, nil
	}

	anchorText := strings.TrimSpace(cfg.AnchorDate)
	if anchorText == "" {
		return Policy{}, fmt.Errorf("season lifecycle anchor date is required when enabled")
	}
	anchor, err := time.ParseInLocation(time.DateOnly, anchorText, location)
	if err != nil {
		return Policy{}, fmt.Errorf("parse season lifecycle anchor date: %w", err)
	}
	if anchor.Day() != 1 {
		return Policy{}, fmt.Errorf("season lifecycle anchor date must be the first day of a month")
	}
	policy.Anchor = anchor
	return policy, nil
}

func LocationForConfig(cfg config.SeasonLifecycleConfig) (*time.Location, error) {
	policy, err := NewPolicy(cfg)
	if err != nil {
		return nil, err
	}
	return policy.Location, nil
}

func (p Policy) StartedAt(now time.Time) bool {
	return p.Enabled && !now.In(p.Location).Before(p.Anchor)
}

func (p Policy) WindowAt(now time.Time) (Window, bool) {
	if !p.StartedAt(now) {
		return Window{}, false
	}
	localNow := now.In(p.Location)
	months := (localNow.Year()-p.Anchor.Year())*12 + int(localNow.Month()-p.Anchor.Month())
	index := months / p.CycleMonths
	start := p.Anchor.AddDate(0, index*p.CycleMonths, 0)
	endExclusive := start.AddDate(0, p.CycleMonths, 0)
	return Window{
		Number:    p.InitialNumber + index,
		Name:      fmt.Sprintf("S%d", p.InitialNumber+index),
		StartDate: start,
		EndDate:   endExclusive.AddDate(0, 0, -1),
	}, true
}

func (p Policy) WindowsThrough(now time.Time, futureWindows int) []Window {
	current, ok := p.WindowAt(now)
	if !ok {
		return []Window{}
	}
	if futureWindows < 0 {
		futureWindows = 0
	}
	count := current.Number - p.InitialNumber + futureWindows + 1
	windows := make([]Window, 0, count)
	for index := 0; index < count; index++ {
		start := p.Anchor.AddDate(0, index*p.CycleMonths, 0)
		windows = append(windows, Window{
			Number:    p.InitialNumber + index,
			Name:      fmt.Sprintf("S%d", p.InitialNumber+index),
			StartDate: start,
			EndDate:   start.AddDate(0, p.CycleMonths, 0).AddDate(0, 0, -1),
		})
	}
	return windows
}

func Resolve(policy Policy, now time.Time, existing []model.Season) Resolution {
	plan := BuildPlan(policy, now, existing)
	result := Resolution{State: plan.State, Plan: plan}
	if len(plan.Conflicts) > 0 {
		result.Problem = strings.Join(plan.Conflicts, "; ")
	}
	if plan.State != StateActive || plan.Current == nil {
		return result
	}
	for index := range existing {
		if MatchesWindow(existing[index], *plan.Current, policy.Location) {
			item := existing[index]
			result.Season = &item
			return result
		}
	}
	result.State = StateUnavailable
	result.Problem = "current season window is missing"
	return result
}

func WindowSeason(window Window) model.Season {
	return model.Season{
		Name:      window.Name,
		StartDate: window.StartDate,
		EndDate:   window.EndDate,
	}
}

func BoundsForConfig(cfg config.SeasonLifecycleConfig, season *model.Season) (time.Time, time.Time, error) {
	location, err := LocationForConfig(cfg)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	start, endExclusive := Bounds(season, location)
	return start, endExclusive, nil
}

func Bounds(season *model.Season, location *time.Location) (time.Time, time.Time) {
	if season == nil {
		return time.Time{}, time.Time{}
	}
	if location == nil {
		location = season.StartDate.Location()
		if location == nil {
			location = time.UTC
		}
	}
	start := dateAtLocation(season.StartDate, location)
	return start, dateAtLocation(season.EndDate, location).AddDate(0, 0, 1)
}

func Contains(season *model.Season, at time.Time, location *time.Location) bool {
	start, endExclusive := Bounds(season, location)
	if start.IsZero() || endExclusive.IsZero() {
		return false
	}
	return !at.Before(start) && at.Before(endExclusive)
}

func BuildPlan(policy Policy, now time.Time, existing []model.Season) Plan {
	if !policy.Enabled || !policy.StartedAt(now) {
		return Plan{State: StateNotStarted, Windows: []Window{}, Missing: []Window{}, Conflicts: []string{}}
	}

	windows := policy.WindowsThrough(now, 1)
	plan := Plan{Windows: windows, Missing: []Window{}, Conflicts: []string{}}
	if len(windows) == 0 {
		plan.State = StateUnavailable
		plan.Conflicts = append(plan.Conflicts, "unable to derive current season window")
		return plan
	}
	current := windows[len(windows)-2]
	plan.Current = &current

	expectedByStart := make(map[string]Window, len(windows))
	for _, window := range windows {
		expectedByStart[dateKey(window.StartDate, policy.Location)] = window
	}
	lastWindow := windows[len(windows)-1]
	matching := make(map[string]model.Season, len(windows))
	relevant := make([]model.Season, 0, len(existing))
	for _, item := range existing {
		start, endExclusive := Bounds(&item, policy.Location)
		if endExclusive.Before(policy.Anchor) || endExclusive.Equal(policy.Anchor) || start.After(lastWindow.EndDate) {
			continue
		}
		relevant = append(relevant, item)
		key := dateKey(start, policy.Location)
		window, expected := expectedByStart[key]
		if !expected {
			plan.Conflicts = append(plan.Conflicts, fmt.Sprintf("unexpected season %q starts on %s", item.Name, key))
			continue
		}
		if _, duplicate := matching[key]; duplicate {
			plan.Conflicts = append(plan.Conflicts, fmt.Sprintf("duplicate season start date %s", key))
			continue
		}
		if !sameWindow(item, window, policy.Location) {
			plan.Conflicts = append(plan.Conflicts, fmt.Sprintf("season %q conflicts with deterministic window %s", item.Name, window.Name))
			continue
		}
		matching[key] = item
	}

	sort.Slice(relevant, func(i, j int) bool {
		left, _ := Bounds(&relevant[i], policy.Location)
		right, _ := Bounds(&relevant[j], policy.Location)
		return left.Before(right)
	})
	for index := 1; index < len(relevant); index++ {
		previousStart, previousEnd := Bounds(&relevant[index-1], policy.Location)
		currentStart, _ := Bounds(&relevant[index], policy.Location)
		if previousEnd.After(currentStart) {
			plan.Conflicts = append(plan.Conflicts, fmt.Sprintf("seasons starting %s and %s overlap", dateKey(previousStart, policy.Location), dateKey(currentStart, policy.Location)))
		}
	}

	for _, window := range windows {
		if _, found := matching[dateKey(window.StartDate, policy.Location)]; !found {
			plan.Missing = append(plan.Missing, window)
		}
	}
	if len(plan.Conflicts) > 0 || containsWindow(plan.Missing, current, policy.Location) {
		plan.State = StateUnavailable
		return plan
	}
	plan.State = StateActive
	return plan
}

func MatchesWindow(existing model.Season, window Window, location *time.Location) bool {
	start, endExclusive := Bounds(&existing, location)
	wantEndExclusive := window.EndDate.AddDate(0, 0, 1)
	return existing.Name == window.Name && start.Equal(window.StartDate) && endExclusive.Equal(wantEndExclusive)
}

func sameWindow(existing model.Season, window Window, location *time.Location) bool {
	return MatchesWindow(existing, window, location)
}

func containsWindow(windows []Window, target Window, location *time.Location) bool {
	for _, item := range windows {
		if dateKey(item.StartDate, location) == dateKey(target.StartDate, location) {
			return true
		}
	}
	return false
}

func dateAtLocation(value time.Time, location *time.Location) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, location)
}

func dateKey(value time.Time, location *time.Location) string {
	return dateAtLocation(value, location).Format(time.DateOnly)
}
