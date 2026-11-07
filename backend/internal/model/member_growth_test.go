package model

import (
	"errors"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newMemberGrowthTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&MemberGrowthProfile{}, &MemberGrowthLog{}); err != nil {
		t.Fatalf("prepare member growth schema: %v", err)
	}
	return db
}

func TestMemberGrowthModelsExposeExpectedTableNames(t *testing.T) {
	if got := (MemberGrowthProfile{}).TableName(); got != "member_growth_profiles" {
		t.Fatalf("expected member growth profile table name, got %s", got)
	}
	if got := (MemberGrowthLog{}).TableName(); got != "member_growth_logs" {
		t.Fatalf("expected member growth log table name, got %s", got)
	}
}

func TestMemberGrowthLogUniqueConstraintRejectsDuplicateMatchGrant(t *testing.T) {
	db := newMemberGrowthTestDB(t)
	model := NewMemberGrowthLogModel(db)
	first := &MemberGrowthLog{
		UserId:       1001,
		MatchId:      18,
		GrowthPoints: 1,
		Source:       MemberGrowthSourceRealMatchCompleted,
	}
	if err := model.Create(first); err != nil {
		t.Fatalf("create first growth log: %v", err)
	}

	err := model.Create(&MemberGrowthLog{
		UserId:       1001,
		MatchId:      18,
		GrowthPoints: 1,
		Source:       MemberGrowthSourceRealMatchCompleted,
	})
	if err == nil {
		t.Fatal("expected duplicate growth log create to fail")
	}
}

func TestMemberGrowthProfileModelFindOrCreateAndUpdate(t *testing.T) {
	db := newMemberGrowthTestDB(t)
	model := NewMemberGrowthProfileModel(db)

	profile, err := model.FindOrCreate(1001)
	if err != nil {
		t.Fatalf("find or create profile: %v", err)
	}
	if profile.UserId != 1001 || profile.GrowthLevel != 1 || profile.GrowthPoints != 0 {
		t.Fatalf("unexpected initial profile: %+v", profile)
	}

	day := time.Date(2026, 4, 11, 0, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	profile.GrowthPoints = 10
	profile.GrowthLevel = 2
	profile.TodayGrowthCount = 3
	profile.TodayGrowthDate = &day
	if err := model.Update(profile); err != nil {
		t.Fatalf("update profile: %v", err)
	}

	reloaded, err := model.FindByUserId(1001)
	if err != nil {
		t.Fatalf("reload profile: %v", err)
	}
	if reloaded == nil {
		t.Fatal("expected reloaded profile")
	}
	if reloaded.GrowthPoints != 10 || reloaded.GrowthLevel != 2 || reloaded.TodayGrowthCount != 3 {
		t.Fatalf("unexpected reloaded profile: %+v", reloaded)
	}
	if reloaded.TodayGrowthDate == nil || !reloaded.TodayGrowthDate.Equal(day) {
		t.Fatalf("unexpected today growth date: %+v", reloaded.TodayGrowthDate)
	}
}

func TestMemberGrowthLogModelFindByUserMatchAndSource(t *testing.T) {
	db := newMemberGrowthTestDB(t)
	model := NewMemberGrowthLogModel(db)
	if err := model.Create(&MemberGrowthLog{
		UserId:       1001,
		MatchId:      88,
		GrowthPoints: 1,
		Source:       MemberGrowthSourceRealMatchCompleted,
	}); err != nil {
		t.Fatalf("create growth log: %v", err)
	}

	log, err := model.FindByUserMatchAndSource(1001, 88, MemberGrowthSourceRealMatchCompleted)
	if err != nil {
		t.Fatalf("find growth log: %v", err)
	}
	if log == nil || log.MatchId != 88 {
		t.Fatalf("unexpected growth log: %+v", log)
	}

	missing, err := model.FindByUserMatchAndSource(1001, 99, MemberGrowthSourceRealMatchCompleted)
	if err != nil {
		t.Fatalf("find missing growth log: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected missing growth log, got %+v", missing)
	}
}

func TestMemberGrowthModelsReturnHelpfulNilErrors(t *testing.T) {
	var profileModel *MemberGrowthProfileModel
	if _, err := profileModel.FindByUserId(1); !errors.Is(err, ErrMemberGrowthProfileDBNil) {
		t.Fatalf("expected nil db error, got %v", err)
	}

	var logModel *MemberGrowthLogModel
	if _, err := logModel.FindByUserMatchAndSource(1, 1, MemberGrowthSourceRealMatchCompleted); !errors.Is(err, ErrMemberGrowthLogDBNil) {
		t.Fatalf("expected nil db error, got %v", err)
	}
}

func TestFindGrowthPointsByUserIdsReturnsOnlyExistingProfiles(t *testing.T) {
	db := newMemberGrowthTestDB(t)
	profileModel := NewMemberGrowthProfileModel(db)
	// 1100 个档案跨过 500 条一批的分批边界。
	const total = 1100
	profiles := make([]MemberGrowthProfile, 0, total)
	ids := make([]int64, 0, total+1)
	for i := 1; i <= total; i++ {
		profiles = append(profiles, MemberGrowthProfile{UserId: int64(i), GrowthPoints: i * 2, GrowthLevel: 1})
		ids = append(ids, int64(i))
	}
	ids = append(ids, total+1)
	if err := db.CreateInBatches(&profiles, 200).Error; err != nil {
		t.Fatalf("seed profiles: %v", err)
	}

	got, err := profileModel.FindGrowthPointsByUserIds(ids)
	if err != nil {
		t.Fatalf("find growth points: %v", err)
	}
	if len(got) != total || got[1] != 2 || got[501] != 1002 || got[total] != total*2 {
		t.Fatalf("unexpected growth points: len=%d sample=%d/%d/%d", len(got), got[1], got[501], got[total])
	}
	if _, ok := got[total+1]; ok {
		t.Fatal("user without a profile must stay out of the result")
	}

	empty, err := profileModel.FindGrowthPointsByUserIds(nil)
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty ids: %v err=%v", empty, err)
	}
}
