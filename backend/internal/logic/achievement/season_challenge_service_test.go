package achievement

import (
	"context"
	"fmt"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newSeasonChallengeTestService(t *testing.T) (*SeasonChallengeService, *svc.ServiceContext, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.AchievementProgressEvent{}, &model.SeasonChallengeSnapshot{}); err != nil {
		t.Fatalf("prepare schema: %v", err)
	}
	ctx := &svc.ServiceContext{
		DB:                            db,
		AchievementProgressEventModel: model.NewAchievementProgressEventModel(db),
		SeasonChallengeSnapshotModel:  model.NewSeasonChallengeSnapshotModel(db),
	}
	return NewSeasonChallengeService(ctx), ctx, db
}

func TestFixedSeasonChallengeDefinitions(t *testing.T) {
	definitions := FixedSeasonChallengeDefinitions()
	if len(definitions) != 3 {
		t.Fatalf("expected three definitions, got %d", len(definitions))
	}
	wants := map[string]int{
		SeasonChallengeMatchesKey:    20,
		SeasonChallengeWinsKey:       10,
		SeasonChallengeTournamentKey: 1,
	}
	for _, definition := range definitions {
		if wants[definition.Key] != definition.Threshold {
			t.Fatalf("unexpected definition: %+v", definition)
		}
	}
}

func TestSeasonChallengeServiceReturnsZeroProgressWithoutEvents(t *testing.T) {
	service, _, _ := newSeasonChallengeTestService(t)
	season := testSeason(3)

	list, err := service.GetProgress(10, &season, 3)
	if err != nil {
		t.Fatalf("get progress: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected three challenges, got %d", len(list))
	}
	for _, item := range list {
		if item.Progress != 0 || item.Completed {
			t.Fatalf("expected zero incomplete challenge: %+v", item)
		}
	}
}

func TestSeasonChallengeServiceAggregatesBySeasonUserAndGameType(t *testing.T) {
	service, ctx, _ := newSeasonChallengeTestService(t)
	season := testSeason(3)
	inside := season.StartDate.Add(24 * time.Hour)
	_, endExclusive, err := service.seasonBounds(&season)
	if err != nil {
		t.Fatalf("resolve season bounds: %v", err)
	}
	after := endExclusive.Add(time.Hour)

	events := []*model.AchievementProgressEvent{
		model.NewAchievementProgressEvent(10, SourceTypeMatch, 1, 3, MetricMatchesTotal, 12, inside),
		model.NewAchievementProgressEvent(10, SourceTypeMatch, 2, 3, MetricWinsTotal, 7, inside),
		model.NewAchievementProgressEvent(10, SourceTypeTournamentFinish, 3, 3, MetricTournamentFinishTotal, 1, inside),
		model.NewAchievementProgressEvent(10, SourceTypeMatch, 4, 2, MetricMatchesTotal, 99, inside),
		model.NewAchievementProgressEvent(11, SourceTypeMatch, 5, 3, MetricMatchesTotal, 99, inside),
		model.NewAchievementProgressEvent(10, SourceTypeMatch, 6, 3, MetricMatchesTotal, 99, after),
	}
	for _, event := range events {
		if _, err := ctx.AchievementProgressEventModel.CreateIfAbsent(event); err != nil {
			t.Fatalf("append event: %v", err)
		}
	}

	list, err := service.GetProgress(10, &season, 3)
	if err != nil {
		t.Fatalf("get progress: %v", err)
	}
	byKey := challengeProgressByKey(list)
	if byKey[SeasonChallengeMatchesKey].Progress != 12 || byKey[SeasonChallengeMatchesKey].Completed {
		t.Fatalf("unexpected match challenge: %+v", byKey[SeasonChallengeMatchesKey])
	}
	if byKey[SeasonChallengeWinsKey].Progress != 7 || byKey[SeasonChallengeWinsKey].Completed {
		t.Fatalf("unexpected win challenge: %+v", byKey[SeasonChallengeWinsKey])
	}
	if byKey[SeasonChallengeTournamentKey].Progress != 1 || !byKey[SeasonChallengeTournamentKey].Completed {
		t.Fatalf("unexpected tournament challenge: %+v", byKey[SeasonChallengeTournamentKey])
	}
}

func TestSeasonChallengeServiceUsesEndExclusiveBoundary(t *testing.T) {
	service, ctx, _ := newSeasonChallengeTestService(t)
	season := testSeason(3)
	_, endExclusive, err := service.seasonBounds(&season)
	if err != nil {
		t.Fatalf("resolve season bounds: %v", err)
	}
	for _, event := range []*model.AchievementProgressEvent{
		model.NewAchievementProgressEvent(10, SourceTypeMatch, 100, 3, MetricMatchesTotal, 1, endExclusive.Add(-time.Second)),
		model.NewAchievementProgressEvent(10, SourceTypeMatch, 101, 3, MetricMatchesTotal, 99, endExclusive),
	} {
		if _, err := ctx.AchievementProgressEventModel.CreateIfAbsent(event); err != nil {
			t.Fatalf("seed boundary event: %v", err)
		}
	}
	progress, err := service.GetProgress(10, &season, 3)
	if err != nil {
		t.Fatalf("get boundary progress: %v", err)
	}
	if progress[0].Progress != 1 {
		t.Fatalf("only end-date event before the next boundary may count: %+v", progress)
	}
}

func TestSeasonChallengeArchiveUsesBoundedGroupedQueries(t *testing.T) {
	for _, testCase := range []struct {
		pairs       int
		wantQueries int64
	}{{pairs: 1, wantQueries: 2}, {pairs: 100, wantQueries: 2}, {pairs: 1200, wantQueries: 6}} {
		t.Run(fmt.Sprintf("pairs_%d", testCase.pairs), func(t *testing.T) {
			_, _, db := newSeasonChallengeTestService(t)
			season := testSeason(3)
			events := make([]model.AchievementProgressEvent, 0, testCase.pairs)
			for index := 1; index <= testCase.pairs; index++ {
				events = append(events, *model.NewAchievementProgressEvent(int64(index), SourceTypeMatch, int64(index), 3, MetricMatchesTotal, 1, season.StartDate.Add(time.Hour)))
			}
			if err := db.CreateInBatches(&events, 200).Error; err != nil {
				t.Fatalf("seed archive events: %v", err)
			}
			metrics := observability.NewRequestMetrics(time.Now())
			requestDB := db.Session(&gorm.Session{Logger: observability.NewGormLogger(time.Hour)}).
				WithContext(observability.WithRequestMetrics(context.Background(), metrics))
			requestSvc := &svc.ServiceContext{
				DB:                            requestDB,
				AchievementProgressEventModel: model.NewAchievementProgressEventModel(requestDB),
				SeasonChallengeSnapshotModel:  model.NewSeasonChallengeSnapshotModel(requestDB),
			}
			snapshots, err := NewSeasonChallengeService(requestSvc).BuildSnapshotsWithTx(nil, &season, season.EndDate.AddDate(0, 0, 1))
			if err != nil || len(snapshots) != testCase.pairs*3 {
				t.Fatalf("build grouped snapshots: count=%d err=%v", len(snapshots), err)
			}
			if sqlCount := metrics.Snapshot().SQLCount; sqlCount != testCase.wantQueries {
				t.Fatalf("unexpected grouped archive query count: got=%d want=%d", sqlCount, testCase.wantQueries)
			}
		})
	}
}

func TestSeasonChallengeServiceArchivesThreeReadOnlySnapshotsPerUserGame(t *testing.T) {
	service, ctx, db := newSeasonChallengeTestService(t)
	season := testSeason(3)
	inside := season.StartDate.Add(24 * time.Hour)
	for _, event := range []*model.AchievementProgressEvent{
		model.NewAchievementProgressEvent(10, SourceTypeMatch, 1, 3, MetricMatchesTotal, 20, inside),
		model.NewAchievementProgressEvent(10, SourceTypeMatch, 1, 3, MetricWinsTotal, 10, inside),
		model.NewAchievementProgressEvent(10, SourceTypeTournamentFinish, 2, 3, MetricTournamentFinishTotal, 1, inside),
		model.NewAchievementProgressEvent(20, SourceTypeMatch, 3, 2, MetricMatchesTotal, 2, inside),
	} {
		if _, err := ctx.AchievementProgressEventModel.CreateIfAbsent(event); err != nil {
			t.Fatalf("append event: %v", err)
		}
	}

	archivedAt := season.EndDate.AddDate(0, 0, 1)
	first, err := service.ArchiveSeasonWithTx(nil, &season, archivedAt)
	if err != nil {
		t.Fatalf("archive season: %v", err)
	}
	if len(first) != 6 {
		t.Fatalf("expected six snapshots for two user/game pairs, got %d", len(first))
	}
	if _, err := service.ArchiveSeasonWithTx(nil, &season, archivedAt); err != nil {
		t.Fatalf("archive season again: %v", err)
	}

	var total int64
	if err := db.Model(&model.SeasonChallengeSnapshot{}).Count(&total).Error; err != nil {
		t.Fatalf("count snapshots: %v", err)
	}
	if total != 6 {
		t.Fatalf("expected idempotent six rows, got %d", total)
	}

	list, err := service.FindArchived(10, season.Id, 3)
	if err != nil {
		t.Fatalf("find archived: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected three archived challenges, got %d", len(list))
	}
	for _, item := range list {
		if item.Completed != 1 {
			t.Fatalf("expected completed snapshot: %+v", item)
		}
	}
}

func testSeason(id int64) model.Season {
	return model.Season{
		Id:        id,
		Name:      "S3",
		StartDate: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
		Status:    1,
	}
}

func challengeProgressByKey(list []SeasonChallengeProgress) map[string]SeasonChallengeProgress {
	result := make(map[string]SeasonChallengeProgress, len(list))
	for _, item := range list {
		result[item.Key] = item
	}
	return result
}
