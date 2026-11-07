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

func TestDropMatchRankSettlementStateMigrationIsGuardedAndReversible(t *testing.T) {
	content, err := os.ReadFile("20261015170000_drop_match_rank_settlement_state.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	up, down, found := strings.Cut(string(content), "-- +goose Down")
	if !found || !strings.HasPrefix(up, "-- +goose Up") {
		t.Fatal("migration must contain goose Up then Down")
	}
	// Up 只删除异议期留下的列与索引，且每一步都先查 information_schema，可重复执行。
	for _, snippet := range []string{
		"information_schema.COLUMNS",
		"information_schema.STATISTICS",
		"DROP COLUMN `rank_excluded`",
		"DROP INDEX `idx_matches_rank_settlement`",
		"DROP COLUMN `rank_settlement_due_at`",
		"DROP COLUMN `rank_settlement_status`",
	} {
		if !strings.Contains(up, snippet) {
			t.Fatalf("Up missing %q", snippet)
		}
	}
	if strings.Contains(up, "ADD ") {
		t.Fatal("Up must only drop the dispute-window columns")
	}
	// Down 恢复列、索引与仅限约球比赛的回填，供回滚到异议期版本。
	for _, snippet := range []string{
		"ADD COLUMN `rank_settlement_status`",
		"ADD COLUMN `rank_settlement_due_at`",
		"ADD INDEX `idx_matches_rank_settlement`",
		"ADD COLUMN `rank_excluded`",
		"WHERE `challenge_id` IS NOT NULL",
	} {
		if !strings.Contains(down, snippet) {
			t.Fatalf("Down missing %q", snippet)
		}
	}
	if strings.Contains(down, "DROP ") {
		t.Fatal("Down must only restore the dropped schema")
	}
	if strings.Contains(string(content), "ADD COLUMN IF NOT EXISTS") || strings.Contains(string(content), "DROP COLUMN IF EXISTS") {
		t.Fatal("migration must remain compatible with older MySQL")
	}
}
