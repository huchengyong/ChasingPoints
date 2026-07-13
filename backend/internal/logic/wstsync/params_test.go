package wstsync

import "testing"

func TestParseSyncParamsSeasonMode(t *testing.T) {
	params, err := ParseSyncParams([]string{"--season", "2025"})
	if err != nil {
		t.Fatalf("parse params: %v", err)
	}

	if params.Mode != SyncModeSeason {
		t.Fatalf("expected season mode, got %s", params.Mode)
	}
	if params.Season != 2025 {
		t.Fatalf("expected season 2025, got %d", params.Season)
	}
	if !params.Publish {
		t.Fatal("expected publish to default to true")
	}
	if !params.IncludeQualifiers {
		t.Fatal("expected include qualifiers to default to true")
	}
	if params.GameType != 1 {
		t.Fatalf("expected default game type 1, got %d", params.GameType)
	}
}

func TestParseSyncParamsYearMode(t *testing.T) {
	params, err := ParseSyncParams([]string{"--year", "2025"})
	if err != nil {
		t.Fatalf("parse params: %v", err)
	}

	if params.Mode != SyncModeYear {
		t.Fatalf("expected year mode, got %s", params.Mode)
	}
	if params.Year != 2025 {
		t.Fatalf("expected year 2025, got %d", params.Year)
	}
	if params.From == nil || params.To == nil {
		t.Fatal("expected year mode to resolve a full window")
	}
	if got := params.From.Format("2006-01-02"); got != "2025-01-01" {
		t.Fatalf("expected from 2025-01-01, got %s", got)
	}
	if got := params.To.Format("2006-01-02"); got != "2025-12-31" {
		t.Fatalf("expected to 2025-12-31, got %s", got)
	}
}

func TestParseSyncParamsRangeMode(t *testing.T) {
	params, err := ParseSyncParams([]string{"--from", "2026-01-01", "--to", "2026-04-30"})
	if err != nil {
		t.Fatalf("parse params: %v", err)
	}

	if params.Mode != SyncModeRange {
		t.Fatalf("expected range mode, got %s", params.Mode)
	}
	if params.From == nil || params.To == nil {
		t.Fatal("expected range mode to set both window bounds")
	}
	if got := params.From.Format("2006-01-02"); got != "2026-01-01" {
		t.Fatalf("expected from 2026-01-01, got %s", got)
	}
	if got := params.To.Format("2006-01-02"); got != "2026-04-30" {
		t.Fatalf("expected to 2026-04-30, got %s", got)
	}
}

func TestParseSyncParamsRejectsMixedModes(t *testing.T) {
	_, err := ParseSyncParams([]string{"--season", "2025", "--year", "2025"})
	if err == nil {
		t.Fatal("expected mixed modes to fail")
	}
}

func TestParseSyncParamsRejectsIncompleteRange(t *testing.T) {
	_, err := ParseSyncParams([]string{"--from", "2026-01-01"})
	if err == nil {
		t.Fatal("expected incomplete range to fail")
	}
}
