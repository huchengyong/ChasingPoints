package wstsync

import (
	"fmt"
	"strings"
	"time"
)

const dateOnlyLayout = "2006-01-02"

var qualifierKeywords = []string{
	"qualifier",
	"qualifiers",
	"qualification",
	"preliminary",
}

func parseDateOnly(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}

	parsed, err := time.ParseInLocation(dateOnlyLayout, strings.TrimSpace(value), time.UTC)
	if err != nil {
		return nil, err
	}
	normalized := parsed.UTC()
	return &normalized, nil
}

func buildYearWindow(year int) (DateWindow, error) {
	if year <= 0 {
		return DateWindow{}, fmt.Errorf("year must be positive")
	}

	from := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(year, time.December, 31, 0, 0, 0, 0, time.UTC)
	return DateWindow{From: from, To: to}, nil
}

func normalizeDateOnly(t time.Time) time.Time {
	if t.IsZero() {
		return time.Time{}
	}
	return time.Date(t.UTC().Year(), t.UTC().Month(), t.UTC().Day(), 0, 0, 0, 0, time.UTC)
}

func (w DateWindow) Contains(t time.Time) bool {
	if w.Empty() {
		return true
	}

	value := normalizeDateOnly(t)
	from := normalizeDateOnly(w.From)
	to := normalizeDateOnly(w.To)

	if !from.IsZero() && value.Before(from) {
		return false
	}
	if !to.IsZero() && value.After(to) {
		return false
	}
	return true
}

func (w DateWindow) Overlaps(start, end time.Time) bool {
	if w.Empty() {
		return true
	}

	start = normalizeDateOnly(start)
	end = normalizeDateOnly(end)
	from := normalizeDateOnly(w.From)
	to := normalizeDateOnly(w.To)

	if end.IsZero() {
		end = start
	}
	if start.IsZero() {
		start = end
	}
	if !from.IsZero() && end.Before(from) {
		return false
	}
	if !to.IsZero() && start.After(to) {
		return false
	}
	return true
}

func IsQualifierTournament(name string) bool {
	lowered := strings.ToLower(strings.TrimSpace(name))
	if lowered == "" {
		return false
	}

	for _, keyword := range qualifierKeywords {
		if strings.Contains(lowered, keyword) {
			return true
		}
	}
	return false
}

func SelectTournamentsForWindow(items []TournamentResource, window DateWindow, includeQualifiers bool) []TournamentResource {
	selected := make([]TournamentResource, 0, len(items))
	for _, item := range items {
		if !includeQualifiers && IsQualifierTournament(item.Attributes.Name) {
			continue
		}

		startAt, err := parseDateOnly(item.Attributes.StartDate)
		if err != nil {
			continue
		}
		endAt, err := parseDateOnly(item.Attributes.EndDate)
		if err != nil {
			continue
		}

		start := time.Time{}
		end := time.Time{}
		if startAt != nil {
			start = *startAt
		}
		if endAt != nil {
			end = *endAt
		}
		if !window.Overlaps(start, end) {
			continue
		}

		selected = append(selected, item)
	}

	return selected
}

func FilterMatchesByTournamentIDs(items []MatchResource, tournamentIDs map[string]struct{}) []MatchResource {
	if len(tournamentIDs) == 0 || len(items) == 0 {
		return []MatchResource{}
	}

	filtered := make([]MatchResource, 0, len(items))
	for _, item := range items {
		if _, ok := tournamentIDs[item.Attributes.TournamentID]; !ok {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}
