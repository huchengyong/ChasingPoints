package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestPlayerNameZhMigrationIsCompatibleAndKeepsDefaultCharset(t *testing.T) {
	content, err := os.ReadFile("20261112170000_add_player_name_zh.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	text := string(content)
	for _, snippet := range []string{
		"information_schema.COLUMNS",
		"ADD COLUMN `name_zh` VARCHAR(64) NOT NULL DEFAULT ''''",
		"-- +goose Down",
		"DROP COLUMN `name_zh`",
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("migration missing %q", snippet)
		}
	}
	upper := strings.ToUpper(text)
	for _, forbidden := range []string{"IF NOT EXISTS", "IF EXISTS", "CHARSET", "CHARACTER SET", "COLLATE"} {
		if strings.Contains(upper, forbidden) {
			t.Fatalf("migration must not contain %q", forbidden)
		}
	}
}
