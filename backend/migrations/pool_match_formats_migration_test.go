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

func TestMatchRankSettlementStateMigrationIsCompatibleAndBoundedBackfill(t *testing.T) {
	content, err := os.ReadFile("20261015100000_add_match_rank_settlement_state.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	text := string(content)
	for _, snippet := range []string{
		"information_schema.COLUMNS",
		"information_schema.STATISTICS",
		"`rank_settlement_status`",
		"`rank_settlement_due_at`",
		"idx_matches_rank_settlement",
		"match_participant_results",
		"`rank_excluded`",
		"DEFAULT ''none''",
		"-- +goose Down",
		"DROP INDEX `idx_matches_rank_settlement`",
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("migration missing %q", snippet)
		}
	}
	// 回填只能落在约球关联比赛上（走 challenge_id 唯一索引），不得全表扫描更新 matches。
	if !strings.Contains(text, "WHERE `challenge_id` IS NOT NULL") {
		t.Fatal("backfill must be restricted to challenge-linked matches")
	}
	if strings.Contains(text, "ADD COLUMN IF NOT EXISTS") || strings.Contains(text, "DROP COLUMN IF EXISTS") {
		t.Fatal("migration must remain compatible with older MySQL")
	}
}
