package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestPoolMatchFormatsMigrationIsCompatibleAndLeavesLegacyDefault(t *testing.T) {
	content, err := os.ReadFile("20260819100000_add_pool_match_formats.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	text := string(content)
	for _, snippet := range []string{
		"information_schema.COLUMNS",
		"`match_format`",
		"`target_wins`",
		"DEFAULT ''legacy''",
		"DEFAULT 0",
		"-- +goose Down",
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("migration missing %q", snippet)
		}
	}
	if strings.Contains(text, "UPDATE `matches`") {
		t.Fatal("pool match format migration must not backfill existing matches")
	}
	if strings.Contains(text, "ADD COLUMN IF NOT EXISTS") || strings.Contains(text, "DROP COLUMN IF EXISTS") {
		t.Fatal("migration must remain compatible with older MySQL")
	}
}
