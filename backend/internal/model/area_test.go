package model

import (
	"testing"

	"chasing_points/internal/testsupport"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAreaModelFindChildrenByParentID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := testsupport.PrepareAreaSchema(db); err != nil {
		t.Fatalf("prepare area schema: %v", err)
	}

	rows := []map[string]any{
		{"area_id": 19, "parent_id": 0, "name": "广东省"},
		{"area_id": 20, "parent_id": 0, "name": "广西壮族自治区"},
		{"area_id": 321, "parent_id": 19, "name": "深圳市"},
		{"area_id": 322, "parent_id": 19, "name": "广州市"},
	}
	for _, row := range rows {
		if err := db.Table("dou_area").Create(row).Error; err != nil {
			t.Fatalf("seed dou_area row: %v", err)
		}
	}

	model := NewAreaModel(db)
	list, err := model.FindChildren(19)
	if err != nil {
		t.Fatalf("find children: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 children, got %d", len(list))
	}
	if list[0].AreaId != 321 || list[0].Name != "深圳市" {
		t.Fatalf("unexpected first child %#v", list[0])
	}
	if list[1].AreaId != 322 || list[1].ParentId != 19 {
		t.Fatalf("unexpected second child %#v", list[1])
	}
}
