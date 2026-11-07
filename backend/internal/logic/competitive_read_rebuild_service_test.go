package logic

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/model"
)

func TestCompetitiveReadRebuildBackfillsBeforeProjectingAndCanPauseResume(t *testing.T) {
	db, ctx := newCompetitiveProjectorTestContext(t)
	if err := db.AutoMigrate(&model.Match{}); err != nil {
		t.Fatalf("prepare match schema: %v", err)
	}
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "one"}, {Id: 2, Nickname: "two"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	draw := 3
	base := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	matches := []model.Match{
		{Id: 1, UserId: 1, OpponentId: ptrInt64(2), OpponentName: "two", GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &draw, MatchTime: base, EndTime: &base},
		{Id: 2, UserId: 1, OpponentId: ptrInt64(2), OpponentName: "two", GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &draw, MatchTime: base.Add(time.Hour), EndTime: ptrTime(base.Add(time.Hour))},
	}
	if err := db.Create(&matches).Error; err != nil {
		t.Fatalf("seed matches: %v", err)
	}

	service := NewCompetitiveReadModelRebuildService(ctx)
	summary, err := service.Rebuild(context.Background(), CompetitiveReadModelRebuildOptions{BatchSize: 1, MaxBatches: 1})
	if err != nil {
		t.Fatalf("backfill first batch: %v", err)
	}
	if summary.Batches != 0 || summary.BackfillBatches != 1 || summary.CompletionTimesBackfilled != 1 {
		t.Fatalf("first run must only backfill the first source batch: %+v", summary)
	}
	checkpoint, err := ctx.CompetitiveReadModel.FindCheckpoint(competitiveReadModelRebuildJobName)
	if err != nil || checkpoint == nil || checkpoint.BackfillCursorMatchId != 1 || checkpoint.BackfillCompleted {
		t.Fatalf("checkpoint after first backfill batch: %+v err=%v", checkpoint, err)
	}

	if err := service.Pause(); err != nil {
		t.Fatalf("pause rebuild: %v", err)
	}
	paused, err := service.Rebuild(context.Background(), CompetitiveReadModelRebuildOptions{BatchSize: 1})
	if err != nil || !paused.Paused || paused.MatchesScanned != 0 {
		t.Fatalf("paused rebuild must not scan: %+v err=%v", paused, err)
	}
	if err := service.Resume(); err != nil {
		t.Fatalf("resume rebuild: %v", err)
	}
	summary, err = service.Rebuild(context.Background(), CompetitiveReadModelRebuildOptions{BatchSize: 1})
	if err != nil || summary.LastCursorMatchID != 2 || !summary.CompletedAtBackfillComplete {
		t.Fatalf("resume rebuild must complete the ordered projection: %+v err=%v", summary, err)
	}

	first, err := ctx.CompetitiveReadModel.FindParticipantByMatchAndUser(1, 1)
	if err != nil || first == nil {
		t.Fatalf("first projected draw: %+v err=%v", first, err)
	}
	for _, matchID := range []int64{1, 2} {
		var stored model.Match
		if err := db.First(&stored, matchID).Error; err != nil || stored.CompletedAt == nil {
			t.Fatalf("historical completed_at must be backfilled: match=%d value=%+v err=%v", matchID, stored, err)
		}
	}
}

func TestCompetitiveReadRebuildDryRunDoesNotWriteProjectionOrCheckpoint(t *testing.T) {
	db, ctx := newCompetitiveProjectorTestContext(t)
	if err := db.AutoMigrate(&model.Match{}); err != nil {
		t.Fatalf("prepare match schema: %v", err)
	}
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "one"}, {Id: 2, Nickname: "two"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	result := 1
	completedAt := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	if err := db.Create(&model.Match{
		Id: 3, UserId: 1, OpponentId: ptrInt64(2), OpponentName: "two", GameType: 3,
		MatchMode: model.MatchModeRanked, Status: 2, Result: &result, MatchTime: completedAt, EndTime: &completedAt,
	}).Error; err != nil {
		t.Fatalf("seed completed match: %v", err)
	}

	summary, err := NewCompetitiveReadModelRebuildService(ctx).DryRun(context.Background(), CompetitiveReadModelRebuildOptions{BatchSize: 1})
	if err != nil || !summary.DryRun || summary.MatchesScanned != 1 || summary.ParticipantRows != 2 {
		t.Fatalf("unexpected dry-run summary: %+v err=%v", summary, err)
	}
	projection, err := ctx.CompetitiveReadModel.FindParticipantByMatchAndUser(3, 1)
	if err != nil || projection != nil {
		t.Fatalf("dry run must not write participant projection: %+v err=%v", projection, err)
	}
	checkpoint, err := ctx.CompetitiveReadModel.FindCheckpoint(competitiveReadModelRebuildJobName)
	if err != nil || checkpoint != nil {
		t.Fatalf("dry run must not write checkpoint: %+v err=%v", checkpoint, err)
	}
	var stored model.Match
	if err := db.First(&stored, 3).Error; err != nil || stored.CompletedAt != nil {
		t.Fatalf("dry run must not backfill completed_at: %+v err=%v", stored, err)
	}
}

func TestCompetitiveReadAuditOnlyReportsDifferences(t *testing.T) {
	db, ctx := newCompetitiveProjectorTestContext(t)
	if err := db.AutoMigrate(&model.Match{}, &model.MatchRound{}); err != nil {
		t.Fatalf("prepare audit schema: %v", err)
	}
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "one"}, {Id: 2, Nickname: "two"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	win := 1
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	match := model.Match{Id: 10, UserId: 1, OpponentId: ptrInt64(2), OpponentName: "two", GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &win, MatchTime: now, EndTime: &now, CompletedAt: &now}
	if err := db.Create(&match).Error; err != nil {
		t.Fatalf("seed audit match: %v", err)
	}
	if err := db.Create(&model.MatchRound{MatchId: 10, RoundNo: 1, Winner: ptrInt(1), WinType: "normal"}).Error; err != nil {
		t.Fatalf("seed audit round: %v", err)
	}

	summary, err := NewCompetitiveReadAuditService(ctx).Audit(context.Background(), 10)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if len(summary.Differences) == 0 {
		t.Fatal("audit should report missing projections without repairing them")
	}
	projection, err := ctx.CompetitiveReadModel.FindParticipantByMatchAndUser(10, 1)
	if err != nil || projection != nil {
		t.Fatalf("audit must not write missing projection: %+v err=%v", projection, err)
	}
}

func TestCompetitiveReadRebuildRetriesTheFailedMatchFromLastSuccessfulCheckpoint(t *testing.T) {
	db, ctx := newCompetitiveProjectorTestContext(t)
	if err := db.AutoMigrate(&model.Match{}); err != nil {
		t.Fatalf("prepare match schema: %v", err)
	}
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "one"}, {Id: 2, Nickname: "two"}}).Error; err != nil {
		t.Fatalf("seed initial users: %v", err)
	}
	result := 1
	early := time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC)
	late := early.Add(time.Hour)
	matches := []model.Match{
		{Id: 1, UserId: 1, OpponentId: ptrInt64(2), OpponentName: "two", GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &result, MatchTime: early.Add(-time.Minute), EndTime: &early, CompletedAt: &early},
		{Id: 2, UserId: 99, OpponentId: ptrInt64(2), OpponentName: "two", GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &result, MatchTime: late.Add(-time.Minute), EndTime: &late, CompletedAt: &late},
	}
	if err := db.Create(&matches).Error; err != nil {
		t.Fatalf("seed matches: %v", err)
	}

	service := NewCompetitiveReadModelRebuildService(ctx)
	if _, err := service.Rebuild(context.Background(), CompetitiveReadModelRebuildOptions{BatchSize: 10}); err == nil {
		t.Fatal("rebuild must fail for a match whose participant is missing")
	}
	checkpoint, err := ctx.CompetitiveReadModel.FindCheckpoint(competitiveReadModelRebuildJobName)
	if err != nil || checkpoint == nil || checkpoint.CursorMatchId != 1 || checkpoint.CursorCompletedAt == nil || !checkpoint.CursorCompletedAt.Equal(early) {
		t.Fatalf("failed match must not advance checkpoint: %+v err=%v", checkpoint, err)
	}
	if err := db.Create(&model.User{Id: 99, Nickname: "repaired"}).Error; err != nil {
		t.Fatalf("repair missing participant: %v", err)
	}
	if _, err := service.Rebuild(context.Background(), CompetitiveReadModelRebuildOptions{BatchSize: 10}); err != nil {
		t.Fatalf("retry rebuild after repair: %v", err)
	}
	projection, err := ctx.CompetitiveReadModel.FindParticipantByMatchAndUser(2, 99)
	if err != nil || projection == nil {
		t.Fatalf("retry must process the previously failed match: %+v err=%v", projection, err)
	}
}

func TestCompetitiveReadRebuildUsesCompletedAtBeforeMatchIDForStreaks(t *testing.T) {
	db, ctx := newCompetitiveProjectorTestContext(t)
	if err := db.AutoMigrate(&model.Match{}); err != nil {
		t.Fatalf("prepare match schema: %v", err)
	}
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "one"}, {Id: 2, Nickname: "two"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	loss, win := 2, 1
	early := time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC)
	late := early.Add(2 * time.Hour)
	matches := []model.Match{
		{Id: 1, UserId: 1, OpponentId: ptrInt64(2), OpponentName: "two", GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &win, MatchTime: late.Add(-time.Minute), EndTime: &late, CompletedAt: &late},
		{Id: 2, UserId: 1, OpponentId: ptrInt64(2), OpponentName: "two", GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &loss, MatchTime: early.Add(-time.Minute), EndTime: &early, CompletedAt: &early},
	}
	if err := db.Create(&matches).Error; err != nil {
		t.Fatalf("seed out-of-order matches: %v", err)
	}
	if _, err := NewCompetitiveReadModelRebuildService(ctx).Rebuild(context.Background(), CompetitiveReadModelRebuildOptions{BatchSize: 10}); err != nil {
		t.Fatalf("rebuild out-of-order matches: %v", err)
	}
	stats, err := ctx.CompetitiveReadModel.FindStats(1, 0)
	if err != nil || stats == nil || stats.CurrentWinStreak != 1 || stats.MaxWinStreak != 1 || stats.LastMatchId != 1 {
		t.Fatalf("rebuild must replay completion chronology, got %+v err=%v", stats, err)
	}
}

func ptrInt64(value int64) *int64        { return &value }
func ptrTime(value time.Time) *time.Time { return &value }
func ptrInt(value int) *int              { return &value }
