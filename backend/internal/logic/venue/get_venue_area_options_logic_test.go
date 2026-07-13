package venue

import (
	"context"
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetVenueAreaOptionsReturnsChildrenByParentID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := testsupport.PrepareAreaSchema(db); err != nil {
		t.Fatalf("prepare area schema: %v", err)
	}

	rows := []map[string]any{
		{"area_id": 19, "parent_id": 0, "name": "广东省"},
		{"area_id": 321, "parent_id": 19, "name": "深圳市"},
		{"area_id": 322, "parent_id": 19, "name": "广州市"},
		{"area_id": 2723, "parent_id": 321, "name": "南山区"},
	}
	for _, row := range rows {
		if err := db.Table("dou_area").Create(row).Error; err != nil {
			t.Fatalf("seed dou_area row: %v", err)
		}
	}

	logic := NewGetVenueAreaOptionsLogic(context.Background(), &svc.ServiceContext{
		AreaModel: model.NewAreaModel(db),
	})

	resp, err := logic.GetVenueAreaOptions(&types.GetVenueAreaOptionsReq{ParentId: 19})
	if err != nil {
		t.Fatalf("get venue area options: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success response, got %#v", resp)
	}
	if len(resp.List) != 2 {
		t.Fatalf("expected 2 options, got %d", len(resp.List))
	}
	if resp.List[0].AreaId != 321 || resp.List[0].Name != "深圳市" {
		t.Fatalf("unexpected first option %#v", resp.List[0])
	}
	if resp.List[1].AreaId != 322 || resp.List[1].ParentId != 19 {
		t.Fatalf("unexpected second option %#v", resp.List[1])
	}
}
