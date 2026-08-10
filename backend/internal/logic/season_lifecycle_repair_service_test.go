package logic

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/config"
	achievementx "chasing_points/internal/logic/achievement"
	"chasing_points/internal/model"
	seasonx "chasing_points/internal/season"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newSeasonLifecycleRepairTestSvc(t *testing.T, lifecycle config.SeasonLifecycleConfig) *svc.ServiceContext {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Season{}, &model.SeasonSettlement{}, &model.AchievementProgressEvent{}, &model.SeasonChallengeSnapshot{}); err != nil {
		t.Fatalf("migrate repair schema: %v", err)
	}
	return &svc.ServiceContext{
		Config:                        config.Config{SeasonLifecycle: lifecycle},
		DB:                            db,
		SeasonModel:                   model.NewSeasonModel(db),
		SeasonSettlementModel:         model.NewSeasonSettlementModel(db),
		AchievementProgressEventModel: model.NewAchievementProgressEventModel(db),
		SeasonChallengeSnapshotModel:  model.NewSeasonChallengeSnapshotModel(db),
	}
}

func TestSeasonLifecycleRepairDryRunDoesNotWriteAndCurrentWindowUsesExistingEvents(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	svcCtx := newSeasonLifecycleRepairTestSvc(t, config.SeasonLifecycleConfig{Enabled: true, AnchorDate: "2026-08-01"})
	event := model.NewAchievementProgressEvent(10, achievementx.SourceTypeMatch, 1, 3, achievementx.MetricMatchesTotal, 4, time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC))
	if _, err := svcCtx.AchievementProgressEventModel.CreateIfAbsent(event); err != nil {
		t.Fatalf("seed event: %v", err)
	}
	repair := NewSeasonLifecycleRepairService(svcCtx)
	dryRun, err := repair.DryRunAt(context.Background(), now)
	if err != nil {
		t.Fatalf("dry run repair: %v", err)
	}
	if dryRun.State != seasonx.StateUnavailable || dryRun.Problem == "" || dryRun.WindowsPlanned != 2 || len(dryRun.Windows) != 2 || dryRun.EventsCovered != 1 || dryRun.UsersCovered != 1 {
		t.Fatalf("unexpected dry run summary: %+v", dryRun)
	}
	if dryRun.Policy.AnchorDate != "2026-08-01" || dryRun.Windows[0].Name != "S1" || dryRun.Windows[0].Exists || dryRun.Windows[0].Settlement != "not_due" {
		t.Fatalf("dry run must expose policy and planned window details: %+v", dryRun)
	}
	assertSeasonCount(t, svcCtx, 0)

	summary, err := repair.RepairAt(context.Background(), now)
	if err != nil {
		t.Fatalf("repair current gap: %v", err)
	}
	if summary.State != seasonx.StateActive || summary.WindowsCreated != 2 || summary.SeasonsSettled != 0 || !summary.Windows[0].Created || !summary.Windows[1].Created {
		t.Fatalf("unexpected repair summary: %+v", summary)
	}
	assertSeasonCount(t, svcCtx, 2)
	seasons, err := svcCtx.SeasonModel.ListAll()
	if err != nil || len(seasons) != 2 {
		t.Fatalf("list repaired seasons: %+v err=%v", seasons, err)
	}
	if seasons[0].Status != 1 || seasons[1].Status != 0 {
		t.Fatalf("repair must converge current and next statuses: %+v", seasons)
	}
	progress, err := achievementx.NewSeasonChallengeService(svcCtx).GetProgress(10, &seasons[0], 3)
	if err != nil {
		t.Fatalf("get repaired current progress: %v", err)
	}
	if progress[0].Progress != 4 {
		t.Fatalf("existing gap event must immediately count in current challenge: %+v", progress)
	}

	retry, err := repair.RepairAt(context.Background(), now)
	if err != nil || retry.WindowsCreated != 0 || retry.SeasonsSettled != 0 {
		t.Fatalf("repair must be idempotent: %+v err=%v", retry, err)
	}
}

func TestSeasonLifecycleRepairSettlesHistorySilently(t *testing.T) {
	svcCtx := newSeasonSettlementTestSvc(t)
	season, _, now := seedSeasonSettlementScenario(t, svcCtx, false)
	svcCtx.Config.SeasonLifecycle = config.SeasonLifecycleConfig{
		Enabled:       true,
		AnchorDate:    "2026-07-01",
		InitialNumber: 3,
		CycleMonths:   1,
		Timezone:      "Asia/Shanghai",
	}
	repair := NewSeasonLifecycleRepairService(svcCtx)
	summary, err := repair.RepairAt(context.Background(), now)
	if err != nil {
		t.Fatalf("repair historical season: %v", err)
	}
	if summary.State != seasonx.StateActive || summary.WindowsCreated != 2 || summary.SeasonsSettled != 1 || summary.ChallengeSnapshotsWritten != 3 || summary.SeasonRecordsWritten != 3 || summary.SeasonTitlesGranted != 3 {
		t.Fatalf("unexpected historical repair summary: %+v", summary)
	}
	assertSeasonStatus(t, svcCtx, season.Id, 2)
	settlement, err := svcCtx.SeasonSettlementModel.FindBySeasonId(season.Id)
	if err != nil || settlement == nil || settlement.Status != model.SeasonSettlementStatusCompleted {
		t.Fatalf("expected completed historical settlement: %+v err=%v", settlement, err)
	}
	var notifications int64
	if err := svcCtx.DB.Model(&model.Notification{}).Where("type = ?", seasonRolloverNotificationType).Count(&notifications).Error; err != nil {
		t.Fatalf("count silent notifications: %v", err)
	}
	if notifications != 0 {
		t.Fatalf("historical repair must not create notifications, got %d", notifications)
	}

	retry, err := repair.RepairAt(context.Background(), now)
	if err != nil || retry.WindowsCreated != 0 || retry.SeasonsToSettle != 0 || retry.SeasonsSettled != 0 {
		t.Fatalf("historical repair must be idempotent: %+v err=%v", retry, err)
	}
}

func TestSeasonLifecycleRepairReturnsCompletedSummaryWhenSettlementFails(t *testing.T) {
	svcCtx := newSeasonSettlementTestSvc(t)
	season, _, now := seedSeasonSettlementScenario(t, svcCtx, false)
	svcCtx.Config.SeasonLifecycle = config.SeasonLifecycleConfig{
		Enabled:       true,
		AnchorDate:    "2026-07-01",
		InitialNumber: 3,
		CycleMonths:   1,
		Timezone:      "Asia/Shanghai",
	}
	if err := svcCtx.DB.Migrator().DropTable(&model.SeasonChallengeSnapshot{}); err != nil {
		t.Fatalf("drop challenge snapshots: %v", err)
	}

	summary, err := NewSeasonLifecycleRepairService(svcCtx).RepairAt(context.Background(), now)
	if err == nil || summary == nil {
		t.Fatalf("expected repair failure with summary: summary=%+v err=%v", summary, err)
	}
	if summary.WindowsCreated != 2 || len(summary.Failures) != 1 || summary.Failures[0] == "" {
		t.Fatalf("failed repair must retain completed plan and failure details: %+v", summary)
	}
	assertSeasonStatus(t, svcCtx, season.Id, 1)
}

func TestSeasonLifecycleRepairStopsOnScheduleConflict(t *testing.T) {
	svcCtx := newSeasonLifecycleRepairTestSvc(t, config.SeasonLifecycleConfig{Enabled: true, AnchorDate: "2026-08-01"})
	wrong := model.Season{Name: "错误排期", StartDate: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)}
	if err := svcCtx.SeasonModel.Create(&wrong); err != nil {
		t.Fatalf("seed conflict: %v", err)
	}
	repair := NewSeasonLifecycleRepairService(svcCtx)
	summary, err := repair.RepairAt(context.Background(), time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("repair conflict: %v", err)
	}
	if summary.State != seasonx.StateUnavailable || len(summary.Conflicts) == 0 || summary.WindowsCreated != 0 {
		t.Fatalf("expected no-write conflict summary: %+v", summary)
	}
	assertSeasonCount(t, svcCtx, 1)
}
