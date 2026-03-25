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

	var stage EventNewsStage
	if got := stage.TableName(); got != "event_news_stages" {
		t.Fatalf("expected stage table name event_news_stages, got %s", got)
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
	stageModel := NewEventNewsStageModel(db)

	t1 := time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 3, 24, 10, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC)

	first := &EventNews{
		Title:      "斯诺克世锦赛",
		GameType:   1,
		SourceType: "official",
		SourceName: "WST",
		City:       "谢菲尔德",
		Status:     EventNewsStatusLive,
		Featured:   true,
		Published:  true,
		StartTime:  &t1,
		SortTime:   &t1,
	}
	second := &EventNews{
		Title:      "中式八球公开赛",
		GameType:   3,
		SourceType: "manual",
		SourceName: "Admin",
		City:       "北京",
		Status:     EventNewsStatusUpcoming,
		Featured:   false,
		Published:  false,
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
		Featured:   false,
		Published:  true,
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

	if err := stageModel.Create(&EventNewsStage{
		EventId:    first.Id,
		StageName:  "资格赛",
		StageOrder: 10,
		Status:     EventNewsStatusFinished,
		ResultText: "资格赛结束",
		SortTime:   &t1,
	}); err != nil {
		t.Fatalf("create first stage: %v", err)
	}
	if err := stageModel.Create(&EventNewsStage{
		EventId:    first.Id,
		StageName:  "32强",
		StageOrder: 20,
		Status:     EventNewsStatusLive,
		ResultText: "赵心童晋级16强",
		SortTime:   &t2,
	}); err != nil {
		t.Fatalf("create second stage: %v", err)
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
	if list[0].Id != first.Id {
		t.Fatalf("expected featured event first, got id %d", list[0].Id)
	}

	stages, err := stageModel.FindByEventId(first.Id)
	if err != nil {
		t.Fatalf("find stages by event id: %v", err)
	}
	if len(stages) != 2 {
		t.Fatalf("expected 2 stages, got %#v", stages)
	}
	if stages[0].StageOrder != 10 || stages[1].StageOrder != 20 {
		t.Fatalf("expected stages sorted by stage_order, got %#v", stages)
	}

	featured, err := eventModel.FindFeatured()
	if err != nil {
		t.Fatalf("find featured event: %v", err)
	}
	if featured == nil || featured.Id != first.Id {
		t.Fatalf("expected first event as featured, got %#v", featured)
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

	if err := stageModel.SoftDeleteByEventId(first.Id); err != nil {
		t.Fatalf("delete stages by event id: %v", err)
	}
	stages, err = stageModel.FindByEventId(first.Id)
	if err != nil {
		t.Fatalf("refind stages after delete: %v", err)
	}
	if len(stages) != 0 {
		t.Fatalf("expected 0 stages after delete, got %#v", stages)
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
