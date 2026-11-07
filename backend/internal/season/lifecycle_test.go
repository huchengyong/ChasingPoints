package season

import (
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
)

func TestNewPolicyDefaultsAndValidation(t *testing.T) {
	policy, err := NewPolicy(config.SeasonLifecycleConfig{})
	if err != nil {
		t.Fatalf("normalize disabled policy: %v", err)
	}
	if policy.Enabled || policy.InitialNumber != 1 || policy.CycleMonths != 1 || policy.Location.String() != "Asia/Shanghai" {
		t.Fatalf("unexpected defaults: %+v", policy)
	}

	for _, cfg := range []config.SeasonLifecycleConfig{
		{Enabled: true},
		{Enabled: true, AnchorDate: "2026-07-02"},
		{Enabled: true, AnchorDate: "bad-date"},
		{Enabled: true, AnchorDate: "2026-07-01", InitialNumber: -1},
		{Enabled: true, AnchorDate: "2026-07-01", CycleMonths: -1},
		{Enabled: true, AnchorDate: "2026-07-01", Timezone: "Mars/Olympus"},
	} {
		if _, err := NewPolicy(cfg); err == nil {
			t.Fatalf("expected invalid config rejected: %+v", cfg)
		}
	}
}

func TestPolicyGeneratesContinuousCalendarWindows(t *testing.T) {
	policy, err := NewPolicy(config.SeasonLifecycleConfig{
		Enabled:       true,
		AnchorDate:    "2026-11-01",
		InitialNumber: 7,
		CycleMonths:   2,
		Timezone:      "Asia/Shanghai",
	})
	if err != nil {
		t.Fatalf("new policy: %v", err)
	}
	now := time.Date(2027, 1, 31, 23, 59, 0, 0, time.UTC)
	current, ok := policy.WindowAt(now)
	if !ok {
		t.Fatal("expected current window")
	}
	if current.Name != "S8" || current.StartDate.Format(time.DateOnly) != "2027-01-01" || current.EndDate.Format(time.DateOnly) != "2027-02-28" {
		t.Fatalf("unexpected current window: %+v", current)
	}
	windows := policy.WindowsThrough(now, 1)
	if len(windows) != 3 || windows[0].Name != "S7" || windows[1].Name != "S8" || windows[2].Name != "S9" {
		t.Fatalf("unexpected generated windows: %+v", windows)
	}
	if windows[1].EndDate.AddDate(0, 0, 1).Format(time.DateOnly) != windows[2].StartDate.Format(time.DateOnly) {
		t.Fatalf("windows must be contiguous: %+v", windows)
	}
}

func TestWindowSeasonStoresBusinessDatesInUTC(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	item := WindowSeason(Window{
		Name:      "S1",
		StartDate: time.Date(2026, 8, 1, 0, 0, 0, 0, location),
		EndDate:   time.Date(2026, 8, 31, 0, 0, 0, 0, location),
	})
	if item.StartDate.Location() != time.UTC || item.EndDate.Location() != time.UTC ||
		item.StartDate.Format(time.DateOnly) != "2026-08-01" || item.EndDate.Format(time.DateOnly) != "2026-08-31" {
		t.Fatalf("unexpected persisted season dates: %s - %s", item.StartDate, item.EndDate)
	}
}

func TestBoundsUseHalfOpenBusinessWindow(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	item := &model.Season{
		StartDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
	}
	start, endExclusive := Bounds(item, location)
	if start.Format(time.RFC3339) != "2026-07-01T00:00:00+08:00" || endExclusive.Format(time.RFC3339) != "2026-08-01T00:00:00+08:00" {
		t.Fatalf("unexpected bounds: %s - %s", start, endExclusive)
	}
	if !Contains(item, endExclusive.Add(-time.Nanosecond), location) || Contains(item, endExclusive, location) {
		t.Fatal("season bounds must be half-open")
	}
}

func TestBuildPlanFindsMissingWindowsAndConflicts(t *testing.T) {
	policy, err := NewPolicy(config.SeasonLifecycleConfig{
		Enabled:    true,
		AnchorDate: "2026-07-01",
	})
	if err != nil {
		t.Fatalf("new policy: %v", err)
	}
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	plan := BuildPlan(policy, now, nil)
	if plan.State != StateUnavailable || len(plan.Missing) != 3 || plan.Current == nil || plan.Current.Name != "S2" {
		t.Fatalf("unexpected empty plan: %+v", plan)
	}

	windows := policy.WindowsThrough(now, 1)
	existing := make([]model.Season, 0, len(windows))
	for _, window := range windows {
		existing = append(existing, WindowSeason(window))
	}
	plan = BuildPlan(policy, now, existing)
	if plan.State != StateActive || len(plan.Missing) != 0 || len(plan.Conflicts) != 0 {
		t.Fatalf("expected active complete plan: %+v", plan)
	}

	existing[1].Name = "运营手工赛季"
	plan = BuildPlan(policy, now, existing)
	if plan.State != StateUnavailable || len(plan.Conflicts) == 0 {
		t.Fatalf("expected conflicting plan: %+v", plan)
	}
}
