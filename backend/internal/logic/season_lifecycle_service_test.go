package logic

import (
	"context"
	"sync"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	seasonx "chasing_points/internal/season"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newSeasonLifecycleTestSvc(t *testing.T, lifecycle config.SeasonLifecycleConfig) *svc.ServiceContext {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/season_lifecycle.db?_busy_timeout=1000"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Season{}); err != nil {
		t.Fatalf("migrate seasons: %v", err)
	}
	return &svc.ServiceContext{
		Config:      config.Config{SeasonLifecycle: lifecycle},
		DB:          db,
		SeasonModel: model.NewSeasonModel(db),
	}
}

func TestSeasonLifecycleServiceLeavesDisabledAndPreAnchorSchedulesUntouched(t *testing.T) {
	disabled := newSeasonLifecycleTestSvc(t, config.SeasonLifecycleConfig{})
	service := NewSeasonLifecycleService(disabled)
	result, err := service.EnsureAt(context.Background(), time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC))
	if err != nil || result.State != seasonx.StateNotStarted || result.Created != 0 {
		t.Fatalf("unexpected disabled result: %+v err=%v", result, err)
	}
	assertSeasonCount(t, disabled, 0)

	preAnchor := newSeasonLifecycleTestSvc(t, config.SeasonLifecycleConfig{Enabled: true, AnchorDate: "2026-09-01"})
	result, err = NewSeasonLifecycleService(preAnchor).EnsureAt(context.Background(), time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC))
	if err != nil || result.State != seasonx.StateNotStarted || result.Created != 0 {
		t.Fatalf("unexpected pre-anchor result: %+v err=%v", result, err)
	}
	assertSeasonCount(t, preAnchor, 0)
}

func TestSeasonLifecycleServiceCreatesContinuousCurrentAndFutureWindows(t *testing.T) {
	svcCtx := newSeasonLifecycleTestSvc(t, config.SeasonLifecycleConfig{
		Enabled:       true,
		AnchorDate:    "2026-07-01",
		InitialNumber: 3,
		CycleMonths:   1,
		Timezone:      "Asia/Shanghai",
	})
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	service := NewSeasonLifecycleService(svcCtx)
	result, err := service.EnsureAt(context.Background(), now)
	if err != nil {
		t.Fatalf("ensure lifecycle: %v", err)
	}
	if result.State != seasonx.StateActive || result.Created != 3 || result.Season == nil || result.Season.Name != "S4" {
		t.Fatalf("unexpected ensure result: %+v", result)
	}
	seasons, err := svcCtx.SeasonModel.ListAll()
	if err != nil {
		t.Fatalf("list seasons: %v", err)
	}
	if len(seasons) != 3 || seasons[0].Name != "S3" || seasons[1].Name != "S4" || seasons[2].Name != "S5" {
		t.Fatalf("unexpected generated seasons: %+v", seasons)
	}
	if seasons[0].Status != 0 || seasons[1].Status != 0 || seasons[2].Status != 0 {
		t.Fatalf("schedule creation must not advance lifecycle statuses: %+v", seasons)
	}
	if seasons[0].EndDate.AddDate(0, 0, 1).Format(time.DateOnly) != seasons[1].StartDate.Format(time.DateOnly) ||
		seasons[1].EndDate.AddDate(0, 0, 1).Format(time.DateOnly) != seasons[2].StartDate.Format(time.DateOnly) {
		t.Fatalf("seasons must be contiguous: %+v", seasons)
	}

	retry, err := service.EnsureAt(context.Background(), now)
	if err != nil || retry.State != seasonx.StateActive || retry.Created != 0 {
		t.Fatalf("unexpected repeat ensure: %+v err=%v", retry, err)
	}
	assertSeasonCount(t, svcCtx, 3)
}

func TestSeasonLifecycleServicesConvergeAcrossInstances(t *testing.T) {
	svcCtx := newSeasonLifecycleTestSvc(t, config.SeasonLifecycleConfig{Enabled: true, AnchorDate: "2026-07-01"})
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	services := []*SeasonLifecycleService{NewSeasonLifecycleService(svcCtx), NewSeasonLifecycleService(svcCtx)}
	errs := make(chan error, len(services))
	var waitGroup sync.WaitGroup
	for _, service := range services {
		waitGroup.Add(1)
		go func(service *SeasonLifecycleService) {
			defer waitGroup.Done()
			_, err := service.EnsureAt(context.Background(), now)
			errs <- err
		}(service)
	}
	waitGroup.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent ensure failed: %v", err)
		}
	}
	assertSeasonCount(t, svcCtx, 3)
	seasons, err := svcCtx.SeasonModel.ListAll()
	if err != nil {
		t.Fatalf("list concurrent schedules: %v", err)
	}
	for _, item := range seasons {
		if item.Status != 0 {
			t.Fatalf("concurrent schedule creation must not change statuses: %+v", seasons)
		}
	}
}

func TestSeasonLifecycleServiceReportsConflictsWithoutOverwritingRows(t *testing.T) {
	svcCtx := newSeasonLifecycleTestSvc(t, config.SeasonLifecycleConfig{Enabled: true, AnchorDate: "2026-07-01"})
	wrong := model.Season{
		Name:      "手工赛季",
		StartDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
		Status:    1,
	}
	if err := svcCtx.SeasonModel.Create(&wrong); err != nil {
		t.Fatalf("seed conflicting season: %v", err)
	}
	result, err := NewSeasonLifecycleService(svcCtx).EnsureAt(context.Background(), time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ensure conflicting lifecycle: %v", err)
	}
	if result.State != seasonx.StateUnavailable || len(result.Plan.Conflicts) == 0 {
		t.Fatalf("expected unavailable conflict: %+v", result)
	}
	stored, err := svcCtx.SeasonModel.FindById(wrong.Id)
	if err != nil || stored == nil || stored.Name != wrong.Name || stored.Status != wrong.Status {
		t.Fatalf("conflicting row must stay untouched: %+v err=%v", stored, err)
	}
}

func assertSeasonCount(t *testing.T, svcCtx *svc.ServiceContext, want int64) {
	t.Helper()
	var count int64
	if err := svcCtx.DB.Model(&model.Season{}).Count(&count).Error; err != nil {
		t.Fatalf("count seasons: %v", err)
	}
	if count != want {
		t.Fatalf("season count=%d, want %d", count, want)
	}
}
