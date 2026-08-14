package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestCompetitiveReadModelMigrationsDeclareRebuildableSchemaAndIndexes(t *testing.T) {
	completedAt, err := os.ReadFile("20260811100000_add_match_completed_at.sql")
	if err != nil {
		t.Fatalf("read completed-at migration: %v", err)
	}
	for _, snippet := range []string{
		"`completed_at` DATETIME NULL",
		"idx_matches_status_mode_game_completed",
		"idx_matches_completed_at",
		"information_schema.COLUMNS",
		"-- +goose Down",
	} {
		if !strings.Contains(string(completedAt), snippet) {
			t.Fatalf("completed-at migration missing %q", snippet)
		}
	}
	if strings.Contains(string(completedAt), "ADD COLUMN IF NOT EXISTS") {
		t.Fatal("completed-at migration must remain compatible with older MySQL")
	}

	models, err := os.ReadFile("20260811101000_add_competitive_read_models.sql")
	if err != nil {
		t.Fatalf("read competitive model migration: %v", err)
	}
	for _, snippet := range []string{
		"match_participant_results",
		"UNIQUE KEY `uk_match_participant_result` (`match_id`, `user_id`)",
		"user_competitive_stats",
		"UNIQUE KEY `uk_user_competitive_stats_user_game` (`user_id`, `game_type`)",
		"user_opponent_stats",
		"user_opponent_strength_buckets",
		"competitive_read_model_rebuild_checkpoints",
		"DROP TABLE IF EXISTS `match_participant_results`",
	} {
		if !strings.Contains(string(models), snippet) {
			t.Fatalf("competitive model migration missing %q", snippet)
		}
	}

	opponentAggregates, err := os.ReadFile("20260811103000_add_user_opponent_snapshot_aggregates.sql")
	if err != nil {
		t.Fatalf("read opponent aggregate migration: %v", err)
	}
	for _, snippet := range []string{
		"`opponent_name` VARCHAR(128) NOT NULL DEFAULT",
		"`opponent_avatar` VARCHAR(512) NOT NULL DEFAULT",
		"`score_diff_sum` BIGINT NOT NULL DEFAULT 0",
		"`current_win_streak` INT NOT NULL DEFAULT 0",
		"`max_win_streak` INT NOT NULL DEFAULT 0",
		"information_schema.COLUMNS",
		"-- +goose Down",
	} {
		if !strings.Contains(string(opponentAggregates), snippet) {
			t.Fatalf("opponent aggregate migration missing %q", snippet)
		}
	}

	indexes, err := os.ReadFile("20260811102000_add_hot_read_indexes.sql")
	if err != nil {
		t.Fatalf("read hot-read index migration: %v", err)
	}
	for _, snippet := range []string{
		"idx_friend_requests_to_status_id",
		"idx_challenges_status_expires_id",
		"idx_seasons_end_id_status_start",
		"idx_venues_nearby_candidates",
		"idx_status_geo_status",
		"idx_matches_referee_history",
		"idx_matches_public_status_time",
		"-- +goose Down",
	} {
		if !strings.Contains(string(indexes), snippet) {
			t.Fatalf("hot-read index migration missing %q", snippet)
		}
	}

	orderedCheckpoint, err := os.ReadFile("20260811108000_add_ordered_competitive_rebuild_checkpoint.sql")
	if err != nil {
		t.Fatalf("read ordered checkpoint migration: %v", err)
	}
	for _, snippet := range []string{
		"cursor_completed_at",
		"backfill_cursor_match_id",
		"backfill_completed",
		"idx_matches_status_deleted_completed_at_id",
		"-- +goose Down",
	} {
		if !strings.Contains(string(orderedCheckpoint), snippet) {
			t.Fatalf("ordered checkpoint migration missing %q", snippet)
		}
	}

	fallbackIndexes, err := os.ReadFile("20260811109000_add_competitive_profile_fallback_indexes.sql")
	if err != nil {
		t.Fatalf("read competitive profile fallback index migration: %v", err)
	}
	for _, snippet := range []string{
		"idx_matches_user_game_status_mode_score",
		"(`user_id`, `game_type`, `status`, `match_mode`, `my_score`, `completed_at`, `id`)",
		"idx_matches_opponent_game_status_mode_score",
		"(`opponent_id`, `game_type`, `status`, `match_mode`, `opponent_score`, `completed_at`, `id`)",
		"information_schema.STATISTICS",
		"-- +goose Down",
	} {
		if !strings.Contains(string(fallbackIndexes), snippet) {
			t.Fatalf("competitive profile fallback migration missing %q", snippet)
		}
	}
}

func TestPerformanceMigrationsKeepPreparedStatementsGooseCompatible(t *testing.T) {
	for _, path := range []string{
		"20260811100000_add_match_completed_at.sql",
		"20260811102000_add_hot_read_indexes.sql",
		"20260811103000_add_user_opponent_snapshot_aggregates.sql",
		"20260811104000_add_finish_request_expiry_index.sql",
		"20260811105000_add_achievement_sync_pending_index.sql",
		"20260811108000_add_ordered_competitive_rebuild_checkpoint.sql",
		"20260811109000_add_competitive_profile_fallback_indexes.sql",
	} {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(content)
		if strings.Contains(text, "PREPARE stmt FROM @ddl; EXECUTE stmt") || strings.Contains(text, "EXECUTE stmt; DEALLOCATE PREPARE stmt") {
			t.Fatalf("%s combines prepared statements on one line, which Goose sends as invalid MySQL SQL", path)
		}
		if strings.Contains(text, "@idx_exists = 1") {
			t.Fatalf("%s compares STATISTICS row count with one; composite indexes require a > 0 existence check", path)
		}
	}
}
