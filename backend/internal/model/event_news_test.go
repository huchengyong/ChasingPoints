package model

import (
	"testing"
	"time"

	"chasing_points/internal/testsupport"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEventNewsTableNames(t *testing.T) {
	var event EventNews
	if got := event.TableName(); got != "event_news_events" {
		t.Fatalf("expected event table name event_news_events, got %s", got)
	}

	var match TournamentMatch
	if got := match.TableName(); got != "tournament_matches" {
		t.Fatalf("expected tournament match table name tournament_matches, got %s", got)
	}
}

func TestEventNewsModelsCRUDAndQueries(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := testsupport.PrepareEventNewsSchema(db); err != nil {
		t.Fatalf("prepare event news schema: %v", err)
	}

	eventModel := NewEventNewsModel(db)
	tournamentModel := NewTournamentModel(db)
	matchModel := NewTournamentMatchModel(db)

	t1 := time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 3, 24, 10, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC)

	tournament := &Tournament{
		Name:       "斯诺克世锦赛实体",
		GameType:   1,
		Format:     1,
		MaxPlayers: 16,
		Status:     EventNewsStatusLive,
		Country:    "英国",
		City:       "谢菲尔德",
		VenueName:  "Crucible",
		StartDate:  &t1,
		EndDate:    &t2,
		StartTime:  &t1,
	}
	if err := tournamentModel.Create(tournament); err != nil {
		t.Fatalf("create tournament: %v", err)
	}

	first := &EventNews{
		Title:        "斯诺克世锦赛",
		TournamentId: tournament.Id,
		GameType:     1,
		SourceType:   "official",
		SourceName:   "WST",
		City:         "谢菲尔德",
		Status:       EventNewsStatusLive,
		Published:    true,
		StartDate:    &t1,
		StartTime:    &t1,
		SortTime:     &t1,
	}
	second := &EventNews{
		Title:      "中式八球公开赛",
		GameType:   3,
		SourceType: "manual",
		SourceName: "Admin",
		City:       "北京",
		Status:     EventNewsStatusUpcoming,
		Published:  false,
		StartDate:  &t2,
		StartTime:  &t2,
		SortTime:   &t2,
	}
	third := &EventNews{
		Title:      "中式九球大师赛",
		GameType:   2,
		SourceType: "manual",
		SourceName: "Admin",
		City:       "上海",
		Status:     EventNewsStatusFinished,
		Published:  true,
		StartDate:  &t3,
		StartTime:  &t3,
		SortTime:   &t3,
	}

	if err := eventModel.Create(first); err != nil {
		t.Fatalf("create first event: %v", err)
	}
	if err := eventModel.Create(second); err != nil {
		t.Fatalf("create second event: %v", err)
	}
	if err := eventModel.Create(third); err != nil {
		t.Fatalf("create third event: %v", err)
	}

	if err := matchModel.Create(&TournamentMatch{
		TournamentId:   tournament.Id,
		RoundName:      "32强",
		RoundOrder:     20,
		MatchOrder:     1,
		Status:         EventNewsStatusLive,
		StartTime:      &t2,
		HomePlayerName: "赵心童",
		AwayPlayerName: "马克",
		HomeScore:      6,
		AwayScore:      2,
	}); err != nil {
		t.Fatalf("create tournament match: %v", err)
	}

	gotByID, err := eventModel.FindById(first.Id)
	if err != nil {
		t.Fatalf("find event by id: %v", err)
	}
	if gotByID == nil || gotByID.Title != first.Title {
		t.Fatalf("expected first event, got %#v", gotByID)
	}

	list, total, err := eventModel.FindList(1, 10, -1, -1, "", 1)
	if err != nil {
		t.Fatalf("find published event list: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 published events, got %d", total)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(list))
	}
	if list[0].Id != third.Id {
		t.Fatalf("expected latest sort_time event first, got id %d", list[0].Id)
	}

	matches, err := matchModel.FindByTournament(tournament.Id)
	if err != nil {
		t.Fatalf("find matches by tournament id: %v", err)
	}
	if len(matches) != 1 || matches[0].RoundName != "32强" {
		t.Fatalf("expected one match, got %#v", matches)
	}

	updated := *second
	updated.Id = second.Id
	updated.Published = true
	updated.Summary = "已更新摘要"
	if err := eventModel.Update(&updated); err != nil {
		t.Fatalf("update event: %v", err)
	}

	changed, err := eventModel.UpdatePublished(second.Id, true)
	if err != nil {
		t.Fatalf("update published true: %v", err)
	}
	if !changed {
		t.Fatal("expected publish toggle to affect one row")
	}

	refreshed, err := eventModel.FindById(second.Id)
	if err != nil {
		t.Fatalf("refind second event: %v", err)
	}
	if refreshed == nil || !refreshed.Published || refreshed.PublishedAt == nil {
		t.Fatalf("expected published event with published_at, got %#v", refreshed)
	}

	if err := eventModel.UpdateTournamentBinding(second.Id, tournament.Id); err != nil {
		t.Fatalf("bind tournament: %v", err)
	}
	refreshed, err = eventModel.FindById(second.Id)
	if err != nil {
		t.Fatalf("refind second event after tournament binding: %v", err)
	}
	if refreshed == nil || refreshed.TournamentId != tournament.Id {
		t.Fatalf("expected tournament id to be updated, got %#v", refreshed)
	}

	deleted, err := eventModel.SoftDelete(first.Id)
	if err != nil {
		t.Fatalf("soft delete event: %v", err)
	}
	if !deleted {
		t.Fatal("expected soft delete to affect one row")
	}

	afterDelete, err := eventModel.FindById(first.Id)
	if err != nil {
		t.Fatalf("find deleted event: %v", err)
	}
	if afterDelete != nil {
		t.Fatalf("expected soft deleted event to be hidden, got %#v", afterDelete)
	}
}
