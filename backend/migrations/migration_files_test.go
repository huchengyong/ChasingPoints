package migrations

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestAdminTablesMigrationDownDoesNotWrapPlainDropsInStatementBlock(t *testing.T) {
	content, err := os.ReadFile("20260319180000_add_admin_tables.sql")
	if err != nil {
		t.Fatalf("read migration file: %v", err)
	}

	text := string(content)
	downIndex := strings.Index(text, "-- +goose Down")
	if downIndex < 0 {
		t.Fatal("expected goose down block")
	}

	downBlock := text[downIndex:]
	if strings.Contains(downBlock, "-- +goose StatementBegin") {
		t.Fatal("plain DROP TABLE statements in down block should not be wrapped by goose StatementBegin")
	}
}

func TestInitMigrationContainsFinalUserColumns(t *testing.T) {
	content, err := os.ReadFile("1765095195000_init.sql")
	if err != nil {
		t.Fatalf("read init migration: %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "`push_token`") {
		t.Fatal("init migration should include users.push_token in final ddl")
	}
	if !strings.Contains(text, "`member_expires_at`") {
		t.Fatal("init migration should include users.member_expires_at in final ddl")
	}
}

func TestVenueMigrationContainsFinalRewardReviewColumn(t *testing.T) {
	content, err := os.ReadFile("20260303100001_add_tournament_season_venue_notification_rules_tables.sql")
	if err != nil {
		t.Fatalf("read tournament season venue migration: %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "`reject_reason`") {
		t.Fatal("venue baseline migration should include venues.reject_reason in final ddl")
	}
}

func TestFavoriteVenueRewardMigrationDoesNotAlterBaseTables(t *testing.T) {
	content, err := os.ReadFile("20260326173000_add_favorite_venue_reward_tables.sql")
	if err != nil {
		t.Fatalf("read favorite venue reward migration: %v", err)
	}

	text := string(content)
	if strings.Contains(text, "ALTER TABLE `users`") || strings.Contains(text, "ALTER TABLE `venues`") {
		t.Fatal("favorite venue reward migration should be pure create/drop ddl after baseline reset")
	}
	if strings.Contains(text, "PREPARE ") || strings.Contains(text, "information_schema.COLUMNS") {
		t.Fatal("favorite venue reward migration should not include compatibility patch sql after baseline reset")
	}
}

func TestLegacyEventNewsSingleTableMigrationRemoved(t *testing.T) {
	_, err := os.Stat("20260323150000_add_event_news_table.sql")
	if err == nil {
		t.Fatal("legacy single-table event_news migration should be removed from clean baseline")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stat legacy event_news migration: %v", err)
	}
}

func TestEventNewsScheduleMigrationDoesNotRecreateLegacyTableInDown(t *testing.T) {
	content, err := os.ReadFile("20260325110500_replace_event_news_with_schedule_tables.sql")
	if err != nil {
		t.Fatalf("read event news schedule migration: %v", err)
	}

	text := string(content)
	downIndex := strings.Index(text, "-- +goose Down")
	if downIndex < 0 {
		t.Fatal("expected goose down block")
	}
	downBlock := text[downIndex:]
	if strings.Contains(downBlock, "CREATE TABLE IF NOT EXISTS `event_news`") {
		t.Fatal("clean baseline event news migration should not recreate legacy single-table schema in down block")
	}
}

func TestWSTSyncMigrationAddsOfficialSourceFieldsAndIndexes(t *testing.T) {
	content, err := os.ReadFile("20260405100000_add_wst_sync_fields_and_indexes.sql")
	if err != nil {
		t.Fatalf("read wst sync migration: %v", err)
	}

	text := string(content)
	requiredSnippets := []string{
		"`source_tournament_id`",
		"`source_season_id`",
		"`information_page`",
		"`ticketing_link`",
		"`last_synced_at`",
		"`players`",
		"`tournaments`",
		"`tournament_matches`",
		"`source_player_id`",
		"`source_match_id`",
		"UNIQUE",
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(text, snippet) {
			t.Fatalf("expected wst sync migration to contain %q", snippet)
		}
	}
}

func TestWSTSyncJobStatesMigrationCreatesCursorTable(t *testing.T) {
	content, err := os.ReadFile("20260408170000_add_wst_sync_job_states.sql")
	if err != nil {
		t.Fatalf("read wst sync job states migration: %v", err)
	}

	text := string(content)
	requiredSnippets := []string{
		"wst_sync_job_states",
		"`job_name`",
		"`last_successful_sync_at`",
		"UNIQUE KEY `uk_wst_sync_job_states_job_name`",
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(text, snippet) {
			t.Fatalf("expected wst sync job state migration to contain %q", snippet)
		}
	}
}

func TestMemberGrowthMigrationCreatesProfileAndLogTables(t *testing.T) {
	content, err := os.ReadFile("20260411110000_add_member_growth_tables.sql")
	if err != nil {
		t.Fatalf("read member growth migration: %v", err)
	}

	text := string(content)
	requiredSnippets := []string{
		"member_growth_profiles",
		"member_growth_logs",
		"`growth_points`",
		"`growth_level`",
		"`today_growth_count`",
		"`today_growth_date`",
		"`match_id`",
		"`source`",
		"uniq_member_growth_log",
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(text, snippet) {
			t.Fatalf("expected member growth migration to contain %q", snippet)
		}
	}
}

func TestMemberRankingRightsMigrationUpdatesAchievementRewardScores(t *testing.T) {
	content, err := os.ReadFile("20260412193000_update_member_ranking_rights_achievement_scores.sql")
	if err != nil {
		t.Fatalf("read member ranking rights migration: %v", err)
	}

	text := string(content)
	requiredSnippets := []string{
		"achievement_reward_config",
		"'break_50'",
		"'golden_break'",
		"'break_and_run'",
		"'run_out'",
		"'break_100'",
		"'nine_on_break'",
		"'break_147'",
		"8",
		"4",
		"6",
		"16",
		"30",
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(text, snippet) {
			t.Fatalf("expected member ranking rights migration to contain %q", snippet)
		}
	}
}

func TestAdminMemberRightsConfigMigrationCreatesConfigTable(t *testing.T) {
	content, err := os.ReadFile("20260413103000_add_member_rights_config_table.sql")
	if err != nil {
		t.Fatalf("read admin member rights config migration: %v", err)
	}

	text := string(content)
	requiredSnippets := []string{
		"member_rights_configs",
		"`config_key`",
		"`growth_rules_json`",
		"`ranking_rights_rules_json`",
		"`updated_by`",
		"uniq_member_rights_configs_key",
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(text, snippet) {
			t.Fatalf("expected admin member rights config migration to contain %q", snippet)
		}
	}
}

func TestReputationMigrationCreatesConfigProfileAndLogTables(t *testing.T) {
	content, err := os.ReadFile("20260413160000_add_reputation_tables.sql")
	if err != nil {
		t.Fatalf("read reputation migration: %v", err)
	}

	text := string(content)
	requiredSnippets := []string{
		"CREATE TABLE IF NOT EXISTS `reputation_configs`",
		"CREATE TABLE IF NOT EXISTS `user_reputation_profiles`",
		"CREATE TABLE IF NOT EXISTS `user_reputation_logs`",
		"`base_rules_json` JSON NOT NULL",
		"`recovery_rules_json` JSON NOT NULL",
		"`detection_rules_json` JSON NOT NULL",
		"`ban_until` DATETIME DEFAULT NULL",
		"`match_id` BIGINT DEFAULT NULL COMMENT '对局ID，非对局变更为NULL'",
		"UNIQUE KEY `uniq_user_match_change_type_reason_code` (`user_id`, `match_id`, `change_type`, `reason_code`)",
		"DROP TABLE IF EXISTS `user_reputation_logs`",
		"DROP TABLE IF EXISTS `user_reputation_profiles`",
		"DROP TABLE IF EXISTS `reputation_configs`",
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(text, snippet) {
			t.Fatalf("expected reputation migration to contain %q", snippet)
		}
	}

	if strings.Contains(text, "`reputation_score` INT NOT NULL DEFAULT 100") {
		t.Fatal("reputation profile schema should not hardcode a default reputation score")
	}
}

func TestAchievementTitleClosedLoopMigrationContainsClosedLoopSchema(t *testing.T) {
	content, err := os.ReadFile("20260423110000_achievement_title_closed_loop.sql")
	if err != nil {
		t.Fatalf("read achievement title closed loop migration: %v", err)
	}

	text := string(content)
	requiredSnippets := []string{
		"match_achievements",
		"`actor`",
		"uk_match_actor_achievement",
		"`match_id`, `actor`, `achievement_type`",
		"`game_type`",
		"`metric_key`",
		"`progress_mode`",
		"DEFAULT ''sum''",
		"`reward_title_key`",
		"`reward_title_name`",
		"`sort`",
		"`status`",
		"`updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP",
		"`reward_granted`",
		"`reward_granted_at`",
		"`user_achievements` ADD COLUMN `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP",
		"`title_key`",
		"`source_type`",
		"`source_ref_id`",
		"`source_ref_id` BIGINT UNSIGNED NOT NULL DEFAULT 0",
		"`source_ref_name`",
		"`granted_by_achievement_id`",
		"`equipped_at`",
		"`granted_at`",
		"uk_user_title_source",
		"`user_id`, `title_key`, `source_type`, `source_ref_id`",
		"CREATE TABLE IF NOT EXISTS `achievement_progress_events`",
		"`source_id`",
		"`metric_value`",
		"uk_user_source_metric",
		"`user_id`, `source_type`, `source_id`, `metric_key`",
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(text, snippet) {
			t.Fatalf("expected achievement title closed loop migration to contain %q", snippet)
		}
	}

	if strings.Contains(text, "`source_ref_id` BIGINT UNSIGNED DEFAULT NULL") {
		t.Fatal("user_titles.source_ref_id should not stay nullable, otherwise uk_user_title_source cannot deduplicate rows")
	}
}
