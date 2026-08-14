package main

import (
	"path/filepath"
	"strings"
	"testing"

	"chasing_points/internal/logic"
)

func TestRepairCheckpointRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "season-repair.checkpoint")
	if cursor, err := readRepairCheckpoint(path); err != nil || cursor != "" {
		t.Fatalf("missing checkpoint must start empty: cursor=%q err=%v", cursor, err)
	}
	if err := writeRepairCheckpoint(path, "2026-08-01"); err != nil {
		t.Fatalf("write checkpoint: %v", err)
	}
	if cursor, err := readRepairCheckpoint(path); err != nil || cursor != "2026-08-01" {
		t.Fatalf("read checkpoint: cursor=%q err=%v", cursor, err)
	}
}

func TestFormatRepairSummaryIncludesPlanAndActualResults(t *testing.T) {
	output := formatRepairSummary(true, &logic.SeasonLifecycleRepairSummary{
		State:   "active",
		Problem: "",
		Policy: logic.SeasonLifecycleRepairPolicy{
			AnchorDate:    "2026-08-01",
			InitialNumber: 1,
			CycleMonths:   1,
			Timezone:      "Asia/Shanghai",
		},
		Windows: []logic.SeasonLifecycleRepairWindow{{
			Name:       "S1",
			StartDate:  "2026-08-01",
			EndDate:    "2026-08-31",
			SeasonId:   7,
			Status:     1,
			Exists:     true,
			Created:    true,
			Settlement: "not_due",
		}},
		WindowsPlanned:            1,
		WindowsCreated:            1,
		ChallengeSnapshotsWritten: 3,
		SeasonRecordsWritten:      2,
		SeasonTitlesGranted:       2,
		Conflicts:                 []string{"S2 overlaps S3"},
		Failures:                  []string{"write failed"},
	})

	for _, want := range []string{
		"anchor_date=2026-08-01",
		"window: name=S1 start_date=2026-08-01 end_date=2026-08-31",
		"created=true settlement=not_due",
		"conflict: S2 overlaps S3",
		"failure: write failed",
		"challenge_snapshots_written=3",
		"season_records_written=2",
		"season_titles_granted=2",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("repair output missing %q:\n%s", want, output)
		}
	}
}
