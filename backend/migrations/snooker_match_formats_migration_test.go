package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestSnookerMatchFormatsMigrationIsCompatibleAndBackfillsLegacyThresholds(t *testing.T) {
	content, err := os.ReadFile("20260817100000_add_snooker_match_formats.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	text := string(content)
	for _, snippet := range []string{
		"information_schema.COLUMNS",
		"`snooker_format`",
		"`snooker_target_wins`",
		"FLOOR(`best_of_frames` / 2) + 1",
		"`snooker_rules_version` = 2",
		"MOD(`best_of_frames`, 2) = 1",
		"-- +goose Down",
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("migration missing %q", snippet)
		}
	}
	if strings.Contains(text, "ADD COLUMN IF NOT EXISTS") || strings.Contains(text, "DROP COLUMN IF EXISTS") {
		t.Fatal("migration must remain compatible with older MySQL")
	}
}
