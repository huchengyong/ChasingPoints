package model

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newWSTSyncJobStateTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	schema := `
CREATE TABLE wst_sync_job_states (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	job_name TEXT NOT NULL UNIQUE,
	last_successful_sync_at DATETIME NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);`
	if err := db.Exec(schema).Error; err != nil {
		t.Fatalf("create schema: %v", err)
	}

	return db
}

func TestWSTSyncJobStateModelUpsertLastSuccessfulSyncAtCreatesAndUpdates(t *testing.T) {
	db := newWSTSyncJobStateTestDB(t)
	model := NewWSTSyncJobStateModel(db)

	first := time.Date(2026, 4, 8, 1, 2, 3, 0, time.UTC)
	if err := model.UpsertLastSuccessfulSyncAt("auto-sync", first); err != nil {
		t.Fatalf("create state: %v", err)
	}

	state, err := model.FindByJobName("auto-sync")
	if err != nil {
		t.Fatalf("find state after create: %v", err)
	}
	if state == nil || state.LastSuccessfulSyncAt == nil || !state.LastSuccessfulSyncAt.Equal(first) {
		t.Fatalf("expected first sync time %v, got %#v", first, state)
	}

	second := first.Add(12 * time.Hour)
	if err := model.UpsertLastSuccessfulSyncAt("auto-sync", second); err != nil {
		t.Fatalf("update state: %v", err)
	}

	state, err = model.FindByJobName("auto-sync")
	if err != nil {
		t.Fatalf("find state after update: %v", err)
	}
	if state == nil || state.LastSuccessfulSyncAt == nil || !state.LastSuccessfulSyncAt.Equal(second) {
		t.Fatalf("expected updated sync time %v, got %#v", second, state)
	}
}
