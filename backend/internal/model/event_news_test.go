package model

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEventNewsTableName(t *testing.T) {
	var news EventNews
	if got := news.TableName(); got != "event_news" {
		t.Fatalf("expected table name event_news, got %s", got)
	}
}

func TestEventNewsModelCRUDAndQueries(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	model := NewEventNewsModel(db)

	t1 := time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 3, 24, 10, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC)

	first := &EventNews{
		Title:      "斯诺克大师赛",
		GameType:   1,
		SourceType: "official",
		SourceName: "WST",
		City:       "上海",
		Status:     EventNewsStatusUpcoming,
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
		Status:     EventNewsStatusLive,
		Featured:   false,
		Published:  false,
		StartTime:  &t2,
		SortTime:   &t2,
	}
	third := &EventNews{
		Title:      "中式九球邀请赛",
		GameType:   2,
		SourceType: "manual",
		SourceName: "Admin",
		City:       "上海",
		Status:     EventNewsStatusFinished,
		Featured:   true,
		Published:  true,
		StartTime:  &t3,
		SortTime:   &t3,
	}

	if err := model.Create(first); err != nil {
		t.Fatalf("create first: %v", err)
	}
	if err := model.Create(second); err != nil {
		t.Fatalf("create second: %v", err)
	}
	if err := model.Create(third); err != nil {
		t.Fatalf("create third: %v", err)
	}

	gotByID, err := model.FindById(first.Id)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if gotByID == nil || gotByID.Title != first.Title {
		t.Fatalf("expected first record, got %#v", gotByID)
	}

	list, total, err := model.FindList(1, 10, -1, -1, "上海", 1)
	if err != nil {
		t.Fatalf("find list: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 published shanghai rows, got %d", total)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(list))
	}
	if list[0].Id != first.Id {
		t.Fatalf("expected featured item first, got id %d", list[0].Id)
	}

	featured, err := model.FindFeatured()
	if err != nil {
		t.Fatalf("find featured: %v", err)
	}
	if featured == nil || featured.Id != first.Id {
		t.Fatalf("expected featured first record, got %#v", featured)
	}

	updated := *second
	updated.Id = second.Id
	updated.Published = true
	updated.ResultText = "已更新"
	if err := model.Update(&updated); err != nil {
		t.Fatalf("update record: %v", err)
	}

	changed, err := model.UpdatePublished(second.Id, false)
	if err != nil {
		t.Fatalf("update published false: %v", err)
	}
	if !changed {
		t.Fatal("expected update published to affect one row")
	}

	changed, err = model.UpdatePublished(second.Id, true)
	if err != nil {
		t.Fatalf("update published true: %v", err)
	}
	if !changed {
		t.Fatal("expected publish toggle to affect one row")
	}

	refreshed, err := model.FindById(second.Id)
	if err != nil {
		t.Fatalf("refind second: %v", err)
	}
	if refreshed == nil || !refreshed.Published || refreshed.PublishedAt == nil {
		t.Fatalf("expected published record with published_at, got %#v", refreshed)
	}

	deleted, err := model.SoftDelete(first.Id)
	if err != nil {
		t.Fatalf("soft delete: %v", err)
	}
	if !deleted {
		t.Fatal("expected soft delete to affect one row")
	}

	afterDelete, err := model.FindById(first.Id)
	if err != nil {
		t.Fatalf("find deleted: %v", err)
	}
	if afterDelete != nil {
		t.Fatalf("expected soft deleted record to be hidden, got %#v", afterDelete)
	}

	list, total, err = model.FindList(1, 10, -1, -1, "", -1)
	if err != nil {
		t.Fatalf("find list after delete: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 remaining records, got %d", total)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 remaining rows, got %d", len(list))
	}
}
