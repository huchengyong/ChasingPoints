package wstsync

import "testing"

func TestSelectTournamentsForWindowKeepsAllItemsWhenWindowEmpty(t *testing.T) {
	items := []TournamentResource{
		{ID: "t-1", Attributes: TournamentAttributes{Name: "Example Open 2025", StartDate: "2025-01-01", EndDate: "2025-01-07"}},
		{ID: "t-2", Attributes: TournamentAttributes{Name: "Example Masters 2025", StartDate: "2025-02-01", EndDate: "2025-02-07"}},
	}

	selected := SelectTournamentsForWindow(items, DateWindow{}, true)
	if len(selected) != 2 {
		t.Fatalf("expected 2 tournaments, got %d", len(selected))
	}
}

func TestSelectTournamentsForWindowKeepsItemsIntersectingNaturalYear(t *testing.T) {
	window, err := buildYearWindow(2025)
	if err != nil {
		t.Fatalf("build year window: %v", err)
	}

	items := []TournamentResource{
		{ID: "t-1", Attributes: TournamentAttributes{Name: "Dec 2024 Event", StartDate: "2024-12-25", EndDate: "2025-01-03"}},
		{ID: "t-2", Attributes: TournamentAttributes{Name: "Jan 2025 Event", StartDate: "2025-01-20", EndDate: "2025-01-25"}},
		{ID: "t-3", Attributes: TournamentAttributes{Name: "Jan 2026 Event", StartDate: "2026-01-01", EndDate: "2026-01-03"}},
	}

	selected := SelectTournamentsForWindow(items, window, true)
	if len(selected) != 2 {
		t.Fatalf("expected 2 tournaments in 2025 window, got %d", len(selected))
	}
	if selected[0].ID != "t-1" || selected[1].ID != "t-2" {
		t.Fatalf("unexpected selected tournaments: %+v", selected)
	}
}

func TestSelectTournamentsForWindowKeepsItemsIntersectingCustomRange(t *testing.T) {
	from, err := parseDateOnly("2026-01-01")
	if err != nil {
		t.Fatalf("parse from: %v", err)
	}
	to, err := parseDateOnly("2026-04-30")
	if err != nil {
		t.Fatalf("parse to: %v", err)
	}

	items := []TournamentResource{
		{ID: "t-1", Attributes: TournamentAttributes{Name: "March Event", StartDate: "2026-03-10", EndDate: "2026-03-20"}},
		{ID: "t-2", Attributes: TournamentAttributes{Name: "May Event", StartDate: "2026-05-01", EndDate: "2026-05-07"}},
	}

	selected := SelectTournamentsForWindow(items, DateWindow{From: *from, To: *to}, true)
	if len(selected) != 1 || selected[0].ID != "t-1" {
		t.Fatalf("unexpected selected tournaments: %+v", selected)
	}
}

func TestSelectTournamentsForWindowExcludesQualifiersWhenDisabled(t *testing.T) {
	window, err := buildYearWindow(2025)
	if err != nil {
		t.Fatalf("build year window: %v", err)
	}

	items := []TournamentResource{
		{ID: "t-1", Attributes: TournamentAttributes{Name: "Wuhan Open 2025 Qualifiers", StartDate: "2025-06-22", EndDate: "2025-06-24"}},
		{ID: "t-2", Attributes: TournamentAttributes{Name: "Wuhan Open 2025", StartDate: "2025-08-01", EndDate: "2025-08-07"}},
	}

	selected := SelectTournamentsForWindow(items, window, false)
	if len(selected) != 1 || selected[0].ID != "t-2" {
		t.Fatalf("unexpected qualifier-filtered tournaments: %+v", selected)
	}
}

func TestSelectTournamentsForWindowRejectsInvalidDateRanges(t *testing.T) {
	window, _ := buildYearWindow(2025)
	items := []TournamentResource{
		{ID: "missing-start", Attributes: TournamentAttributes{EndDate: "2025-01-02"}},
		{ID: "invalid-end", Attributes: TournamentAttributes{StartDate: "2025-01-01", EndDate: "bad"}},
		{ID: "inverted", Attributes: TournamentAttributes{StartDate: "2025-01-02", EndDate: "2025-01-01"}},
		{ID: "missing-end", Attributes: TournamentAttributes{StartDate: "2025-01-03"}},
	}
	selected := SelectTournamentsForWindow(items, window, true)
	if len(selected) != 1 || selected[0].ID != "missing-end" {
		t.Fatalf("unexpected date-filter result: %+v", selected)
	}
}

func TestFilterMatchesByTournamentIDsKeepsOnlySelectedTournamentIDs(t *testing.T) {
	selectedTournamentIDs := map[string]struct{}{
		"t-1": {},
		"t-3": {},
	}
	items := []MatchResource{
		{ID: "m-1", Attributes: MatchAttributes{TournamentID: "t-1"}},
		{ID: "m-2", Attributes: MatchAttributes{TournamentID: "t-2"}},
		{ID: "m-3", Attributes: MatchAttributes{TournamentID: "t-3"}},
	}

	filtered := FilterMatchesByTournamentIDs(items, selectedTournamentIDs)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 matches after filtering, got %d", len(filtered))
	}
	if filtered[0].ID != "m-1" || filtered[1].ID != "m-3" {
		t.Fatalf("unexpected filtered matches: %+v", filtered)
	}
}
