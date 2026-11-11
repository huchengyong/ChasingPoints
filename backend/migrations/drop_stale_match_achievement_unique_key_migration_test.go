package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestDropStaleMatchAchievementUniqueKeyMigrationDropsOldIndexAndRepairsData(t *testing.T) {
	content, err := os.ReadFile("20261111150000_drop_stale_match_achievement_unique_key.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	text := string(content)
	for _, snippet := range []string{
		"DROP INDEX `uk_match_achievement`",
		"@idx_exists > 0",
		"match_rounds",
		"break_and_run",
		"run_out",
		"golden_break",
		"nine_on_break",
		"-- +goose Down",
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("migration missing %q", snippet)
		}
	}

	upper := strings.ToUpper(text)
	for _, forbidden := range []string{"CHARSET", "CHARACTER SET", "COLLATE"} {
		if strings.Contains(upper, forbidden) {
			t.Fatalf("migration must not specify %s, keep the database default", forbidden)
		}
	}
}

// SQL 里的归一映射必须与 Go 的 normalizeAchievementType 保持一致。
func TestDropStaleMatchAchievementUniqueKeyMigrationNormalizesLikeCode(t *testing.T) {
	content, err := os.ReadFile("20261111150000_drop_stale_match_achievement_unique_key.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	text := string(content)
	for _, snippet := range []string{
		"WHEN 'break_clear' THEN 'break_and_run'",
		"WHEN 'continue_clear' THEN 'run_out'",
		"WHEN 'small_gold' THEN 'golden_break'",
		"WHEN 'big_gold' THEN 'nine_on_break'",
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("migration normalization missing %q", snippet)
		}
	}
}
