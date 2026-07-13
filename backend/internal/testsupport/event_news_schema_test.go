package testsupport

import (
	"testing"
	"time"

	"chasing_points/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPrepareEventNewsSchemaCreatesUsableTables(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	if err := PrepareEventNewsSchema(db); err != nil {
		t.Fatalf("prepare event news schema: %v", err)
	}

	eventModel := model.NewEventNewsModel(db)
	tournamentModel := model.NewTournamentModel(db)
	matchModel := model.NewTournamentMatchModel(db)
	now := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)

	tournament := &model.Tournament{
		Name:       "测试赛事实体",
		GameType:   1,
		Format:     1,
		MaxPlayers: 16,
		Status:     model.EventNewsStatusUpcoming,
		StartTime:  &now,
	}
	if err := tournamentModel.Create(tournament); err != nil {
		t.Fatalf("create tournament after schema prepare: %v", err)
	}

	event := &model.EventNews{
		Title:        "测试赛事",
		TournamentId: tournament.Id,
		GameType:     1,
		SourceType:   "manual",
		SourceName:   "Admin",
		Status:       model.EventNewsStatusUpcoming,
		SortTime:     &now,
	}
	if err := eventModel.Create(event); err != nil {
		t.Fatalf("create event after schema prepare: %v", err)
	}

	if err := matchModel.Create(&model.TournamentMatch{
		TournamentId:   tournament.Id,
		RoundName:      "资格赛",
		RoundOrder:     10,
		MatchOrder:     1,
		StartTime:      &now,
		Status:         model.EventNewsStatusUpcoming,
		HomePlayerName: "选手A",
		AwayPlayerName: "选手B",
	}); err != nil {
		t.Fatalf("create match after schema prepare: %v", err)
	}
}
