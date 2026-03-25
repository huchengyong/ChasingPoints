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
	stageModel := model.NewEventNewsStageModel(db)
	now := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)

	event := &model.EventNews{
		Title:      "测试赛事",
		GameType:   1,
		SourceType: "manual",
		SourceName: "Admin",
		Status:     model.EventNewsStatusUpcoming,
		SortTime:   &now,
	}
	if err := eventModel.Create(event); err != nil {
		t.Fatalf("create event after schema prepare: %v", err)
	}

	if err := stageModel.Create(&model.EventNewsStage{
		EventId:    event.Id,
		StageName:  "资格赛",
		StageOrder: 10,
		Status:     model.EventNewsStatusUpcoming,
		SortTime:   &now,
	}); err != nil {
		t.Fatalf("create stage after schema prepare: %v", err)
	}
}
