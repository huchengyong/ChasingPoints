package wstsync

import (
	"testing"
	"time"

	"chasing_points/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newWSTSyncTournamentTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.Tournament{}); err != nil {
		t.Fatalf("migrate tournaments: %v", err)
	}
	return db
}

func TestUpsertTournamentsInsertsMissingTournament(t *testing.T) {
	db := newWSTSyncTournamentTestDB(t)
	now := time.Date(2026, 4, 5, 11, 0, 0, 0, time.UTC)
	startDate := time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC)

	err := UpsertTournaments(db, now, []TournamentUpsertRecord{
		{
			SourceType:         "official",
			SourceTournamentId: "tour-1",
			SourceSeasonId:     "2025",
			Name:               "Sportsbet.io Tour Championship 2026",
			CoverImage:         "https://images.gc.wstservices.co.uk/fit-in/400x600/test.png",
			GameType:           1,
			Status:             model.EventNewsStatusFinished,
			Country:            "England",
			City:               "Manchester",
			VenueName:          "Manchester Central",
			StartDate:          &startDate,
			EndDate:            &endDate,
			InformationPage:    "https://www.wst.tv/tourchampionship/",
			TicketingLink:      "https://tickets.example.com",
		},
	})
	if err != nil {
		t.Fatalf("upsert tournaments: %v", err)
	}

	tournament, err := model.NewTournamentModel(db).FindBySourceTournamentIds("official", []string{"tour-1"})
	if err != nil {
		t.Fatalf("find tournament by source id: %v", err)
	}
	got, ok := tournament["tour-1"]
	if !ok {
		t.Fatal("expected inserted tournament")
	}
	if got.Id == 0 {
		t.Fatal("expected inserted tournament to have local id")
	}
	if got.Name != "Sportsbet.io Tour Championship 2026" || got.SourceSeasonId != "2025" {
		t.Fatalf("unexpected inserted tournament: %#v", got)
	}
	if got.Status != model.EventNewsStatusFinished {
		t.Fatalf("expected finished status, got %#v", got)
	}
	if got.LastSyncedAt == nil || !got.LastSyncedAt.Equal(now) {
		t.Fatalf("expected last synced at %v, got %#v", now, got.LastSyncedAt)
	}
}

func TestUpsertTournamentsUpdatesExistingTournamentWithoutChangingID(t *testing.T) {
	db := newWSTSyncTournamentTestDB(t)
	now := time.Date(2026, 4, 5, 11, 0, 0, 0, time.UTC)
	tournamentModel := model.NewTournamentModel(db)
	startDate := time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC)

	seed := &model.Tournament{
		Id:                 77,
		CreatorId:          9,
		Name:               "Old Tournament",
		CoverImage:         "https://example.com/old.png",
		GameType:           1,
		Format:             2,
		MaxPlayers:         32,
		CurrentPlayers:     4,
		Status:             0,
		Country:            "England",
		City:               "Leicester",
		VenueName:          "Old Venue",
		StartDate:          &startDate,
		EndDate:            &endDate,
		SourceType:         "official",
		SourceTournamentId: "tour-1",
		SourceSeasonId:     "2024",
		InformationPage:    "https://example.com/old",
		TicketingLink:      "https://example.com/old-ticket",
	}
	if err := tournamentModel.Create(seed); err != nil {
		t.Fatalf("seed tournament: %v", err)
	}

	if err := UpsertTournaments(db, now, []TournamentUpsertRecord{
		{
			SourceType:         "official",
			SourceTournamentId: "tour-1",
			SourceSeasonId:     "2025",
			Name:               "Sportsbet.io Tour Championship 2026",
			CoverImage:         defaultTournamentCoverImage,
			GameType:           1,
			Status:             model.EventNewsStatusLive,
			Country:            "England",
			City:               "Manchester",
			VenueName:          "",
			StartDate:          &startDate,
			EndDate:            &endDate,
			InformationPage:    "https://www.wst.tv/tourchampionship/",
			TicketingLink:      "https://tickets.example.com",
		},
	}); err != nil {
		t.Fatalf("upsert tournaments: %v", err)
	}

	tournament, err := tournamentModel.FindBySourceTournamentIds("official", []string{"tour-1"})
	if err != nil {
		t.Fatalf("find tournament by source id: %v", err)
	}
	got, ok := tournament["tour-1"]
	if !ok {
		t.Fatal("expected updated tournament")
	}
	if got.Id != 77 {
		t.Fatalf("expected local id to stay 77, got %d", got.Id)
	}
	if got.CreatorId != 9 {
		t.Fatalf("expected creator id to stay 9, got %d", got.CreatorId)
	}
	if got.Name != "Sportsbet.io Tour Championship 2026" || got.SourceSeasonId != "2025" {
		t.Fatalf("unexpected updated tournament: %#v", got)
	}
	if got.Status != model.EventNewsStatusLive {
		t.Fatalf("expected live status after update, got %#v", got)
	}
	if got.CoverImage != "https://example.com/old.png" {
		t.Fatalf("expected existing custom cover to be preserved, got %#v", got)
	}
	if got.VenueName != "Old Venue" {
		t.Fatalf("expected existing venue to be preserved when incoming empty, got %#v", got)
	}
	if got.LastSyncedAt == nil || !got.LastSyncedAt.Equal(now) {
		t.Fatalf("expected last synced at %v, got %#v", now, got.LastSyncedAt)
	}
}
