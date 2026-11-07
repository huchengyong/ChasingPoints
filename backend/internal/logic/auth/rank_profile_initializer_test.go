package auth

import (
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEnsureUserRankingProfilesCreatesAllGameTypesIdempotently(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.UserRanking{}); err != nil {
		t.Fatalf("prepare ranking schema: %v", err)
	}
	svcCtx := &svc.ServiceContext{RankingModel: model.NewRankingModel(db)}
	if err := ensureUserRankingProfiles(svcCtx, 99); err != nil {
		t.Fatalf("initialize rank profiles: %v", err)
	}
	if err := ensureUserRankingProfiles(svcCtx, 99); err != nil {
		t.Fatalf("repeat rank initialization: %v", err)
	}
	rankings, err := svcCtx.RankingModel.ListByUserId(99)
	if err != nil || len(rankings) != len(model.SupportedRankingGameTypes()) {
		t.Fatalf("expected one rank row for each game type: rankings=%#v err=%v", rankings, err)
	}
}
