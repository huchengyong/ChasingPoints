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
