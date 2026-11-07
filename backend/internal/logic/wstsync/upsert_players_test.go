package wstsync

import (
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/testsupport"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newWSTSyncTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := testsupport.PrepareEventNewsSchema(db); err != nil {
		t.Fatalf("prepare schema: %v", err)
	}
	return db
}

func TestUpsertPlayersInsertsMissingPlayer(t *testing.T) {
	db := newWSTSyncTestDB(t)
	now := time.Date(2026, 4, 5, 11, 0, 0, 0, time.UTC)

	err := UpsertPlayers(db, now, []PlayerUpsertRecord{
		{
			SourceType:     "official",
			SourcePlayerId: "player-1",
			FirstName:      "Neil",
			LastName:       "Robertson",
			DisplayName:    "Neil Robertson",
			Avatar:         "https://example.com/neil.png",
			CountryCode:    "gb-aus",
			FlagEmoji:      "🇦🇺",
		},
	})
	if err != nil {
		t.Fatalf("upsert players: %v", err)
	}

	playerMap, err := model.NewPlayerModel(db).FindBySourcePlayerIds("official", []string{"player-1"})
	if err != nil {
		t.Fatalf("find player by source id: %v", err)
	}
	got, ok := playerMap["player-1"]
	if !ok {
		t.Fatal("expected inserted player")
	}
	if got.Id == 0 {
		t.Fatal("expected inserted player to have local id")
	}
	if got.FirstName != "Neil" || got.LastName != "Robertson" || got.Avatar != "https://example.com/neil.png" {
		t.Fatalf("unexpected inserted player: %#v", got)
	}
	if got.FlagEmoji != "🇦🇺" || got.CountryCode != "gb-aus" {
		t.Fatalf("unexpected inserted player nationality fields: %#v", got)
	}
}

func TestUpsertPlayersUpdatesExistingPlayerWithoutChangingID(t *testing.T) {
	db := newWSTSyncTestDB(t)
	now := time.Date(2026, 4, 5, 11, 0, 0, 0, time.UTC)
	playerModel := model.NewPlayerModel(db)

	seed := &model.Player{
		Id:             41,
		SourceType:     "official",
		SourcePlayerId: "player-1",
		FirstName:      "Neil",
		LastName:       "Old",
		DisplayName:    "Neil Old",
		Avatar:         "https://example.com/old.png",
		CountryCode:    "gb-eng",
		FlagEmoji:      "🏴",
	}
	if err := playerModel.Create(seed); err != nil {
		t.Fatalf("seed player: %v", err)
	}

	if err := UpsertPlayers(db, now, []PlayerUpsertRecord{
		{
			SourceType:     "official",
			SourcePlayerId: "player-1",
			FirstName:      "Neil",
			LastName:       "Robertson",
			DisplayName:    "Neil Robertson",
			Avatar:         "https://example.com/new.png",
			CountryCode:    "gb-aus",
			FlagEmoji:      "🇦🇺",
		},
	}); err != nil {
		t.Fatalf("upsert players: %v", err)
	}

	playerMap, err := playerModel.FindBySourcePlayerIds("official", []string{"player-1"})
	if err != nil {
		t.Fatalf("find player by source id: %v", err)
	}
	got, ok := playerMap["player-1"]
	if !ok {
		t.Fatal("expected updated player")
	}
	if got.Id != 41 {
		t.Fatalf("expected local id to stay 41, got %d", got.Id)
	}
	if got.LastName != "Robertson" || got.Avatar != "https://example.com/new.png" || got.CountryCode != "gb-aus" {
		t.Fatalf("unexpected updated player: %#v", got)
	}
	if got.FlagEmoji != "🇦🇺" {
		t.Fatalf("expected flag emoji to be updated from sync source, got %#v", got)
	}
}
