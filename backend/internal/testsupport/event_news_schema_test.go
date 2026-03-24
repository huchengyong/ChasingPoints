package testsupport

import (
	"testing"
	"time"

	"chasing_points/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPrepareEventNewsSchemaCreatesUsableTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	if err := PrepareEventNewsSchema(db); err != nil {
		t.Fatalf("prepare event news schema: %v", err)
	}

	eventNewsModel := model.NewEventNewsModel(db)
	now := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)

	if err := eventNewsModel.Create(&model.EventNews{
		Title:      "测试赛事情报",
		GameType:   1,
		SourceType: "manual",
		SourceName: "Admin",
		Status:     model.EventNewsStatusUpcoming,
		SortTime:   &now,
	}); err != nil {
		t.Fatalf("create event news after schema prepare: %v", err)
	}
}
