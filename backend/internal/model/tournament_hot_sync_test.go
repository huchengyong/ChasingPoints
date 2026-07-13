package model

import (
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTournamentHotSyncTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	schema := `
CREATE TABLE tournaments (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	creator_id INTEGER NOT NULL DEFAULT 0,
	name TEXT NOT NULL,
	description TEXT,
	cover_image TEXT NOT NULL DEFAULT '',
	game_type INTEGER NOT NULL DEFAULT 0,
	format INTEGER NOT NULL DEFAULT 1,
	max_players INTEGER NOT NULL DEFAULT 16,
	current_players INTEGER NOT NULL DEFAULT 0,
	status INTEGER NOT NULL DEFAULT 0,
	country TEXT NOT NULL DEFAULT '',
	city TEXT NOT NULL DEFAULT '',
	venue_name TEXT NOT NULL DEFAULT '',
	start_date DATE NULL,
	end_date DATE NULL,
	start_time DATETIME NULL,
	end_time DATETIME NULL,
	source_type TEXT NOT NULL DEFAULT '',
	source_tournament_id TEXT NOT NULL DEFAULT '',
	source_season_id TEXT NOT NULL DEFAULT '',
	information_page TEXT NOT NULL DEFAULT '',
	ticketing_link TEXT NOT NULL DEFAULT '',
	last_synced_at DATETIME NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);`
	if err := db.Exec(schema).Error; err != nil {
		t.Fatalf("create tournament schema: %v", err)
	}

	return db
}

func TestTournamentModelHasOfficialTournamentsNeedingHotSync(t *testing.T) {
	db := newTournamentHotSyncTestDB(t)
	model := NewTournamentModel(db)

	now := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
	startDate := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	if err := model.Create(&Tournament{
		CreatorId:          0,
		Name:               "Official Live Tournament",
		GameType:           1,
		Status:             EventNewsStatusLive,
		SourceType:         "official",
		SourceTournamentId: "wst-live",
		StartDate:          &startDate,
		EndDate:            &endDate,
	}); err != nil {
		t.Fatalf("create official tournament: %v", err)
	}

	needsHotSync, err := model.HasOfficialTournamentsNeedingHotSync(now, 2, 7)
	if err != nil {
		t.Fatalf("has hot sync targets: %v", err)
	}
	if !needsHotSync {
		t.Fatal("expected hot sync target for live official tournament")
	}
}

func TestTournamentModelHasOfficialTournamentsNeedingHotSyncSkipsIrrelevantRows(t *testing.T) {
	db := newTournamentHotSyncTestDB(t)
	model := NewTournamentModel(db)

	now := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
	oldStart := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
	oldEnd := time.Date(2025, 12, 5, 0, 0, 0, 0, time.UTC)
	if err := model.Create(&Tournament{
		CreatorId:          0,
		Name:               "Old Finished Official Tournament",
		GameType:           1,
		Status:             EventNewsStatusFinished,
		SourceType:         "official",
		SourceTournamentId: "wst-old",
		StartDate:          &oldStart,
		EndDate:            &oldEnd,
	}); err != nil {
		t.Fatalf("create old official tournament: %v", err)
	}
	if err := model.Create(&Tournament{
		CreatorId:          0,
		Name:               "Manual Upcoming Tournament",
		GameType:           1,
		Status:             EventNewsStatusUpcoming,
		SourceType:         "manual",
		SourceTournamentId: "manual-1",
		StartDate:          &oldStart,
		EndDate:            &oldEnd,
	}); err != nil {
		t.Fatalf("create manual tournament: %v", err)
	}

	needsHotSync, err := model.HasOfficialTournamentsNeedingHotSync(now, 2, 7)
	if err != nil {
		t.Fatalf("has hot sync targets: %v", err)
	}
	if needsHotSync {
		t.Fatal("expected no hot sync targets for stale official and manual tournaments")
	}
}
