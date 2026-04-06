package wstsync

import (
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/testsupport"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newProjectionTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := testsupport.PrepareEventNewsSchema(db); err != nil {
		t.Fatalf("prepare event news schema: %v", err)
	}
	return db
}

func TestProjectTournamentEventNewsCreatesOfficialPublishedShell(t *testing.T) {
	db := newProjectionTestDB(t)
	eventNewsModel := model.NewEventNewsModel(db)

	startDate := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 8, 7, 0, 0, 0, 0, time.UTC)
	sortTime := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
	tournament := model.Tournament{
		Id:                 11,
		Name:               "Wuhan Open 2025",
		CoverImage:         "https://example.com/cover.png",
		GameType:           1,
		Status:             model.EventNewsStatusLive,
		Country:            "England",
		City:               "Leicester",
		VenueName:          "Morningside Arena",
		StartDate:          &startDate,
		EndDate:            &endDate,
		SourceType:         "official",
		SourceTournamentId: "wst-11",
		InformationPage:    "https://www.wst.tv/example",
		LastSyncedAt:       &sortTime,
	}
	matches := []model.TournamentMatch{
		{RoundName: "Quarter Finals"},
		{RoundName: "Quarter Finals"},
	}

	projector := NewEventNewsProjector(eventNewsModel)
	item, err := projector.ProjectTournamentEventNews(tournament, matches, true)
	if err != nil {
		t.Fatalf("project event news: %v", err)
	}

	if item.Id <= 0 {
		t.Fatalf("expected persisted event news id, got %+v", item)
	}
	if item.TournamentId != tournament.Id {
		t.Fatalf("expected tournament binding %d, got %d", tournament.Id, item.TournamentId)
	}
	if !item.Published {
		t.Fatalf("expected projected event news to be published: %+v", item)
	}
	if item.SourceType != "official" || item.SourceName != "WST" {
		t.Fatalf("expected official WST source, got %+v", item)
	}
	if item.Title != tournament.Name {
		t.Fatalf("expected title %q, got %q", tournament.Name, item.Title)
	}
	if item.Summary != "共 2 场比赛，当前轮次 Quarter Finals" {
		t.Fatalf("unexpected summary: %q", item.Summary)
	}
}

func TestProjectTournamentEventNewsUpdatesExistingShellWithoutDuplicating(t *testing.T) {
	db := newProjectionTestDB(t)
	eventNewsModel := model.NewEventNewsModel(db)

	startDate := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
	sortTime := time.Date(2025, 9, 1, 12, 0, 0, 0, time.UTC)
	tournament := model.Tournament{
		Id:                 12,
		Name:               "Shanghai Masters 2025",
		CoverImage:         "https://example.com/cover-a.png",
		GameType:           1,
		Status:             model.EventNewsStatusUpcoming,
		Country:            "China",
		City:               "Shanghai",
		VenueName:          "Shanghai Grand Stage",
		StartDate:          &startDate,
		EndDate:            &startDate,
		SourceType:         "official",
		SourceTournamentId: "wst-12",
		InformationPage:    "https://www.wst.tv/shanghai",
		LastSyncedAt:       &sortTime,
	}

	projector := NewEventNewsProjector(eventNewsModel)
	first, err := projector.ProjectTournamentEventNews(tournament, []model.TournamentMatch{{RoundName: "Semi Finals"}}, true)
	if err != nil {
		t.Fatalf("create projection: %v", err)
	}

	tournament.CoverImage = "https://example.com/cover-b.png"
	tournament.Status = model.EventNewsStatusFinished
	second, err := projector.ProjectTournamentEventNews(tournament, []model.TournamentMatch{{RoundName: "Final"}}, true)
	if err != nil {
		t.Fatalf("update projection: %v", err)
	}

	if first.Id != second.Id {
		t.Fatalf("expected stable event news id, got first=%d second=%d", first.Id, second.Id)
	}
	if second.CoverImage != "https://example.com/cover-b.png" {
		t.Fatalf("expected updated cover, got %q", second.CoverImage)
	}
	if second.Status != model.EventNewsStatusFinished {
		t.Fatalf("expected updated status, got %d", second.Status)
	}
	if second.Summary != "共 1 场比赛，当前轮次 Final" {
		t.Fatalf("unexpected updated summary: %q", second.Summary)
	}

	stored, err := eventNewsModel.FindByTournamentAndSourceType(tournament.Id, "official")
	if err != nil {
		t.Fatalf("find by tournament and source type: %v", err)
	}
	if stored == nil || stored.Id != first.Id {
		t.Fatalf("expected one stored projection, got %#v", stored)
	}
}
