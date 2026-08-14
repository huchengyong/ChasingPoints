package logic

import (
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEnsureDefaultRulesContentIsIdempotentStartupInitialization(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.RulesContent{}); err != nil {
		t.Fatalf("prepare rules schema: %v", err)
	}
	svcCtx := &svc.ServiceContext{RulesContentModel: model.NewRulesContentModel(db)}
	if err := EnsureDefaultRulesContent(svcCtx); err != nil {
		t.Fatalf("seed startup rules: %v", err)
	}
	var firstCount int64
	if err := db.Model(&model.RulesContent{}).Count(&firstCount).Error; err != nil || firstCount == 0 {
		t.Fatalf("missing startup rules: count=%d err=%v", firstCount, err)
	}
	if err := EnsureDefaultRulesContent(svcCtx); err != nil {
		t.Fatalf("repeat startup seed: %v", err)
	}
	var secondCount int64
	if err := db.Model(&model.RulesContent{}).Count(&secondCount).Error; err != nil || secondCount != firstCount {
		t.Fatalf("startup seed must be idempotent: first=%d second=%d err=%v", firstCount, secondCount, err)
	}
}
