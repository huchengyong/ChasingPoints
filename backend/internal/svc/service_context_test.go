package svc

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNewServiceModelsWiresAchievementClosedLoopModels(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	models := newServiceModels(db)

	if models.AchievementModel == nil {
		t.Fatal("expected achievement model")
	}
	if models.UserAchievementModel == nil {
		t.Fatal("expected user achievement model")
	}
	if models.UserTitleModel == nil {
		t.Fatal("expected user title model")
	}
	if models.AchievementProgressEventModel == nil {
		t.Fatal("expected achievement progress event model")
	}
}
