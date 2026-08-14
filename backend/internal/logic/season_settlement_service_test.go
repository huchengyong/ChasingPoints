package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	achievementx "chasing_points/internal/logic/achievement"
	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	seasonx "chasing_points/internal/season"
	"chasing_points/internal/svc"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSeasonSettlementServiceSettlesOneSeasonAndIsIdempotent(t *testing.T) {
	svcCtx := newSeasonSettlementTestSvc(t)
	season, nextSeason, now := seedSeasonSettlementScenario(t, svcCtx, true)
	unlockedAt := season.StartDate.Add(10 * 24 * time.Hour)
	equippedAt := unlockedAt.Add(time.Hour)
	careerAchievement := model.Achievement{Id: 900, Key: "match_100", Name: "百战磨砺", Category: "match", GameType: 0, MetricKey: achievementx.MetricMatchesTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 100, Status: 1}
	if err := svcCtx.DB.Create(&careerAchievement).Error; err != nil {
		t.Fatalf("seed career achievement: %v", err)
	}
	if err := svcCtx.DB.Create(&model.UserAchievement{UserId: 10, AchievementId: careerAchievement.Id, Progress: 100, Unlocked: 1, UnlockedAt: &unlockedAt, UnlockedSourceType: achievementx.SourceTypeMatch, UnlockedSourceId: 301, RewardGranted: 1, RewardGrantedAt: &unlockedAt}).Error; err != nil {
		t.Fatalf("seed career progress: %v", err)
	}
	if err := svcCtx.DB.Create(&model.UserTitle{UserId: 10, TitleKey: "title_match_100", TitleName: "资深球手", Source: achievementx.SourceTypeAchievement, SourceType: achievementx.SourceTypeAchievement, SourceRefId: careerAchievement.Id, SourceRefName: careerAchievement.Name, Equipped: 1, EquippedAt: &equippedAt, GrantedAt: &unlockedAt}).Error; err != nil {
		t.Fatalf("seed career title: %v", err)
	}
	miniRedis := miniredis.RunT(t)
	svcCtx.Redis = redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	t.Cleanup(func() { _ = svcCtx.Redis.Close() })
	service := NewSeasonSettlementService(svcCtx)
	service.sendRealtime = func(seasonRolloverRealtimeNotice) error { return nil }

	summary, err := service.SettleSeasonAt(context.Background(), season.Id, now)
	if err != nil {
		t.Fatalf("settle season: %v", err)
	}
	if summary.Skipped || summary.SeasonRecords != 3 || summary.SeasonTitles != 3 || summary.ChallengeSnapshots != 3 || summary.Notifications != 3 {
		t.Fatalf("unexpected settlement summary: %+v", summary)
	}
	if summary.NextSeasonId == nil || *summary.NextSeasonId != nextSeason.Id {
		t.Fatalf("expected next season %d, got %+v", nextSeason.Id, summary.NextSeasonId)
	}
	if version, err := miniRedis.Get("season:info:version"); err != nil || version != "1" {
		t.Fatalf("season rollover must invalidate public season info: version=%q err=%v", version, err)
	}

	assertSeasonStatus(t, svcCtx, season.Id, 2)
	assertSeasonStatus(t, svcCtx, nextSeason.Id, 1)
	assertSingleActiveSeason(t, svcCtx, nextSeason.Id)
	assertSeasonSettlementRecords(t, svcCtx, season.Id)
	assertSeasonSettlementTitles(t, svcCtx, season.Id, 3)
	assertSeasonSettlementSnapshots(t, svcCtx, season.Id, 3)
	assertSeasonRolloverNotifications(t, svcCtx, season, &nextSeason, 3)
	assertCareerAssetsPreserved(t, svcCtx, 10, careerAchievement.Id, unlockedAt, equippedAt)
	challengeProgress, err := achievementx.NewSeasonChallengeService(svcCtx).GetProgress(10, &nextSeason, 3)
	if err != nil {
		t.Fatalf("get next season challenge progress: %v", err)
	}
	for _, item := range challengeProgress {
		if item.Progress != 0 || item.Completed {
			t.Fatalf("next season challenge must start empty: %+v", challengeProgress)
		}
	}

	retrySummary, err := service.SettleSeasonAt(context.Background(), season.Id, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("retry completed settlement: %v", err)
	}
	if !retrySummary.Skipped {
		t.Fatalf("expected completed settlement retry skipped, got %+v", retrySummary)
	}
	assertSeasonSettlementRecords(t, svcCtx, season.Id)
	assertSeasonSettlementTitles(t, svcCtx, season.Id, 3)
	assertSeasonSettlementSnapshots(t, svcCtx, season.Id, 3)
	assertSeasonRolloverNotifications(t, svcCtx, season, &nextSeason, 3)

	settlement, err := svcCtx.SeasonSettlementModel.FindBySeasonId(season.Id)
	if err != nil || settlement == nil {
		t.Fatalf("find completed settlement: settlement=%+v err=%v", settlement, err)
	}
	if settlement.Status != model.SeasonSettlementStatusCompleted || settlement.Attempts != 1 {
		t.Fatalf("unexpected settlement state: %+v", settlement)
	}
}

func TestSeasonSettlementUsesIncrementalRecordsWithoutMatchOrRankHistoryModels(t *testing.T) {
	svcCtx := newSeasonSettlementTestSvc(t)
	season, _, now := seedSeasonSettlementScenario(t, svcCtx, false)
	svcCtx.MatchModel = nil
	svcCtx.RankingModel = nil
	service := NewSeasonSettlementService(svcCtx)
	service.sendRealtime = func(seasonRolloverRealtimeNotice) error { return nil }
	summary, err := service.SettleSeasonWithOptionsAt(context.Background(), season.Id, now, SeasonSettlementOptions{Notify: false})
	if err != nil || summary.SeasonRecords != 3 {
		t.Fatalf("settlement must consume incremental records without match/rank history models: summary=%+v err=%v", summary, err)
	}
	assertSeasonSettlementRecords(t, svcCtx, season.Id)
}

func TestSeasonSettlementRecordReadQueryCountDoesNotGrowWithParticipants(t *testing.T) {
	for _, count := range []int{1, 1000} {
		t.Run(fmt.Sprintf("records_%d", count), func(t *testing.T) {
			svcCtx := newSeasonSettlementTestSvc(t)
			records := make([]model.SeasonRecord, 0, count)
			for index := 1; index <= count; index++ {
				records = append(records, model.SeasonRecord{SeasonId: 88, UserId: int64(index), GameType: 3, EndRankScore: 2000 - index, PeakRankScore: 2000, MatchesPlayed: 1, Wins: index % 2})
			}
			if err := svcCtx.SeasonRecordModel.CreateBatch(records); err != nil {
				t.Fatalf("seed incremental records: %v", err)
			}
			metrics := observability.NewRequestMetrics(time.Now())
			requestDB := svcCtx.DB.Session(&gorm.Session{Logger: observability.NewGormLogger(time.Hour)}).WithContext(observability.WithRequestMetrics(context.Background(), metrics))
			requestSvc := *svcCtx
			requestSvc.SeasonRecordModel = model.NewSeasonRecordModel(requestDB)
			loaded, err := NewSeasonSettlementService(&requestSvc).buildSeasonRecordsWithTx(nil, &model.Season{Id: 88})
			if err != nil || len(loaded) != count {
				t.Fatalf("load settlement records: count=%d err=%v", len(loaded), err)
			}
			if sqlCount := metrics.Snapshot().SQLCount; sqlCount != 1 {
				t.Fatalf("settlement record query count must stay fixed, got %d", sqlCount)
			}
		})
	}
}

func TestSeasonSettlementLargeSeasonUsesBatchedWritesWithoutNPlusOne(t *testing.T) {
	queryCounts := make(map[int]int64)
	for _, recordCount := range []int{10, 2000} {
		t.Run(fmt.Sprintf("records_%d", recordCount), func(t *testing.T) {
			svcCtx := newSeasonSettlementTestSvc(t)
			season := model.Season{Id: 70, Name: "大赛季", StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), Status: 1}
			if err := svcCtx.SeasonModel.Create(&season); err != nil {
				t.Fatalf("create large season: %v", err)
			}
			records := make([]model.SeasonRecord, 0, recordCount)
			for index := 1; index <= recordCount; index++ {
				records = append(records, model.SeasonRecord{SeasonId: season.Id, UserId: int64(index), GameType: 3, StartRankScore: 1000, EndRankScore: 3000 - index, PeakRankScore: 3000, MatchesPlayed: 1, Wins: index % 2})
			}
			if err := svcCtx.SeasonRecordModel.CreateBatch(records); err != nil {
				t.Fatalf("seed large season records: %v", err)
			}
			metrics := observability.NewRequestMetrics(time.Now())
			requestDB := svcCtx.DB.Session(&gorm.Session{Logger: observability.NewGormLogger(time.Hour)}).
				WithContext(observability.WithRequestMetrics(context.Background(), metrics))
			requestSvc := *svcCtx
			requestSvc.DB = requestDB
			requestSvc.UserModel = model.NewUserModel(requestDB)
			requestSvc.MatchModel = model.NewMatchModel(requestDB)
			requestSvc.RankingModel = model.NewRankingModel(requestDB)
			requestSvc.UserTitleModel = model.NewUserTitleModel(requestDB)
			requestSvc.AchievementProgressEventModel = model.NewAchievementProgressEventModel(requestDB)
			requestSvc.NotificationModel = model.NewNotificationModel(requestDB)
			requestSvc.SeasonModel = model.NewSeasonModel(requestDB)
			requestSvc.SeasonRecordModel = model.NewSeasonRecordModel(requestDB)
			requestSvc.SeasonChallengeSnapshotModel = model.NewSeasonChallengeSnapshotModel(requestDB)
			requestSvc.SeasonSettlementModel = model.NewSeasonSettlementModel(requestDB)
			service := NewSeasonSettlementService(&requestSvc)
			service.sendRealtime = func(seasonRolloverRealtimeNotice) error { return nil }
			summary, err := service.SettleSeasonWithOptionsAt(context.Background(), season.Id, time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC), SeasonSettlementOptions{Notify: false})
			if err != nil || summary.SeasonRecords != recordCount || summary.SeasonTitles != minInt(recordCount, 10) {
				t.Fatalf("settle large season: summary=%+v err=%v", summary, err)
			}
			queryCounts[recordCount] = metrics.Snapshot().SQLCount
		})
	}
	if queryCounts[2000] > queryCounts[10]+10 {
		t.Fatalf("large season SQL count must grow only by fixed write batches: small=%d large=%d", queryCounts[10], queryCounts[2000])
	}
}

func TestSeasonSettlementServiceUsesIncrementalRecordsThroughEndDateOnly(t *testing.T) {
	svcCtx := newSeasonSettlementTestSvc(t)
	season := model.Season{
		Id:        30,
		Name:      "S30",
		StartDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
		Status:    1,
	}
	if err := svcCtx.SeasonModel.Create(&season); err != nil {
		t.Fatalf("create boundary season: %v", err)
	}
	for _, userID := range []int64{10, 20} {
		if err := svcCtx.UserModel.Create(&model.User{Id: userID, Nickname: "用户"}); err != nil {
			t.Fatalf("create user %d: %v", userID, err)
		}
	}
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load timezone: %v", err)
	}
	endExclusive := time.Date(2026, 8, 1, 0, 0, 0, 0, location)
	seedSeasonSettlementMatch(t, svcCtx, 3001, 10, 20, 1, endExclusive.Add(-time.Second))
	seedSeasonSettlementMatch(t, svcCtx, 3002, 10, 20, 1, endExclusive)
	if err := svcCtx.SeasonRecordModel.CreateBatch([]model.SeasonRecord{
		{SeasonId: season.Id, UserId: 10, GameType: 3, StartRankScore: 1000, EndRankScore: 1020, PeakRankScore: 1020, MatchesPlayed: 1, Wins: 1},
		{SeasonId: season.Id, UserId: 20, GameType: 3, StartRankScore: 1000, EndRankScore: 980, PeakRankScore: 1000, MatchesPlayed: 1, Wins: 0},
	}); err != nil {
		t.Fatalf("seed boundary season records: %v", err)
	}
	service := NewSeasonSettlementService(svcCtx)
	service.sendRealtime = func(seasonRolloverRealtimeNotice) error { return nil }
	if _, err := service.SettleSeasonAt(context.Background(), season.Id, endExclusive.Add(time.Hour)); err != nil {
		t.Fatalf("settle boundary season: %v", err)
	}
	record, err := svcCtx.SeasonRecordModel.FindBySeasonAndUserAndGameType(season.Id, 10, 3)
	if err != nil || record == nil || record.MatchesPlayed != 1 || record.Wins != 1 {
		t.Fatalf("only completion before end-exclusive boundary may count: %+v err=%v", record, err)
	}
}

func TestSeasonSettlementServiceEntersIntermissionAndLaterActivatesSeason(t *testing.T) {
	svcCtx := newSeasonSettlementTestSvc(t)
	season, _, now := seedSeasonSettlementScenario(t, svcCtx, false)
	service := NewSeasonSettlementService(svcCtx)
	service.sendRealtime = func(seasonRolloverRealtimeNotice) error { return nil }

	summary, err := service.SettleSeasonAt(context.Background(), season.Id, now)
	if err != nil {
		t.Fatalf("settle season without next: %v", err)
	}
	if summary.NextSeasonId != nil {
		t.Fatalf("expected intermission without next season, got %+v", summary.NextSeasonId)
	}
	assertSeasonStatus(t, svcCtx, season.Id, 2)
	assertSingleActiveSeason(t, svcCtx, 0)
	assertSeasonRolloverNotifications(t, svcCtx, season, nil, 3)

	next := model.Season{
		Id:        4,
		Name:      "S4",
		StartDate: now.Add(-time.Hour),
		EndDate:   now.AddDate(0, 1, 0),
		Status:    0,
	}
	if err := svcCtx.SeasonModel.Create(&next); err != nil {
		t.Fatalf("create next season during intermission: %v", err)
	}
	activated, err := service.ActivateReadySeasonAt(context.Background(), now)
	if err != nil {
		t.Fatalf("activate ready season: %v", err)
	}
	if activated == nil || activated.Id != next.Id {
		t.Fatalf("expected season %d activated, got %+v", next.Id, activated)
	}
	assertSingleActiveSeason(t, svcCtx, next.Id)
}

func TestSeasonSettlementDoesNotApplyRankResetRatio(t *testing.T) {
	svcCtx := newSeasonSettlementTestSvc(t)
	season, _, now := seedSeasonSettlementScenario(t, svcCtx, false)
	if err := svcCtx.DB.AutoMigrate(&model.UserRanking{}); err != nil {
		t.Fatalf("migrate ranking snapshot: %v", err)
	}
	if err := svcCtx.DB.Create(&model.UserRanking{UserId: 10, GameType: 3, RankScore: 1680, RankLevel: 5, TotalWins: 12, TotalLosses: 3}).Error; err != nil {
		t.Fatalf("seed ranking: %v", err)
	}
	if err := svcCtx.DB.Model(&model.Season{}).Where("id = ?", season.Id).Update("rank_reset_ratio", 0.2).Error; err != nil {
		t.Fatalf("set rank reset ratio: %v", err)
	}
	service := NewSeasonSettlementService(svcCtx)
	service.sendRealtime = func(seasonRolloverRealtimeNotice) error { return nil }
	if _, err := service.SettleSeasonAt(context.Background(), season.Id, now); err != nil {
		t.Fatalf("settle season: %v", err)
	}
	var ranking model.UserRanking
	if err := svcCtx.DB.Where("user_id = ? AND game_type = ?", 10, 3).First(&ranking).Error; err != nil {
		t.Fatalf("find ranking after settlement: %v", err)
	}
	if ranking.RankScore != 1680 || ranking.RankLevel != 5 {
		t.Fatalf("season settlement must not reset ranking: %+v", ranking)
	}
}

func TestSeasonSettlementServiceCanSettleSilentlyForHistoricalRepair(t *testing.T) {
	svcCtx := newSeasonSettlementTestSvc(t)
	season, _, now := seedSeasonSettlementScenario(t, svcCtx, false)
	service := NewSeasonSettlementService(svcCtx)
	realtimeCalls := 0
	service.sendRealtime = func(seasonRolloverRealtimeNotice) error {
		realtimeCalls++
		return nil
	}

	summary, err := service.SettleSeasonWithOptionsAt(context.Background(), season.Id, now, SeasonSettlementOptions{Notify: false})
	if err != nil {
		t.Fatalf("settle historical season silently: %v", err)
	}
	if summary.Notifications != 0 || realtimeCalls != 0 {
		t.Fatalf("silent settlement must not notify: summary=%+v realtime=%d", summary, realtimeCalls)
	}
	var notifications int64
	if err := svcCtx.DB.Model(&model.Notification{}).Where("type = ?", seasonRolloverNotificationType).Count(&notifications).Error; err != nil {
		t.Fatalf("count silent notifications: %v", err)
	}
	if notifications != 0 {
		t.Fatalf("silent settlement created %d notifications", notifications)
	}
	assertSeasonStatus(t, svcCtx, season.Id, 2)
}

func TestSeasonSettlementServiceUsesSeasonBoundaryForHistoricalTitleGrantTime(t *testing.T) {
	svcCtx := newSeasonSettlementTestSvc(t)
	season, _, now := seedSeasonSettlementScenario(t, svcCtx, false)
	service := NewSeasonSettlementService(svcCtx)
	service.sendRealtime = func(seasonRolloverRealtimeNotice) error { return nil }

	if _, err := service.SettleSeasonWithOptionsAt(context.Background(), season.Id, now.AddDate(0, 2, 0), SeasonSettlementOptions{Notify: false}); err != nil {
		t.Fatalf("settle historical season: %v", err)
	}
	_, expectedGrantedAt, err := seasonx.BoundsForConfig(svcCtx.Config.SeasonLifecycle, &season)
	if err != nil {
		t.Fatalf("resolve season end boundary: %v", err)
	}
	var title model.UserTitle
	if err := svcCtx.DB.Where("source_type = ? AND source_ref_id = ?", achievementx.SourceTypeSeason, season.Id).First(&title).Error; err != nil {
		t.Fatalf("find historical season title: %v", err)
	}
	if title.GrantedAt == nil || !title.GrantedAt.Equal(expectedGrantedAt) {
		t.Fatalf("historical title must use season boundary, got %+v want %s", title.GrantedAt, expectedGrantedAt)
	}
}

func TestSeasonRolloverNotificationsUseBoundedBatchQueries(t *testing.T) {
	for _, testCase := range []struct {
		users       int
		wantQueries int64
	}{{users: 1, wantQueries: 2}, {users: 100, wantQueries: 2}, {users: 1200, wantQueries: 7}} {
		t.Run(fmt.Sprintf("users_%d", testCase.users), func(t *testing.T) {
			svcCtx := newSeasonSettlementTestSvc(t)
			users := make([]model.User, 0, testCase.users)
			records := make([]model.SeasonRecord, 0, testCase.users)
			for index := 1; index <= testCase.users; index++ {
				userID := int64(index)
				users = append(users, model.User{Id: userID, Nickname: fmt.Sprintf("用户%d", index), PushToken: fmt.Sprintf("token-%d", index)})
				records = append(records, model.SeasonRecord{SeasonId: 9, UserId: userID, GameType: 3, FinalRank: index})
			}
			if err := svcCtx.DB.CreateInBatches(&users, 200).Error; err != nil {
				t.Fatalf("seed notification users: %v", err)
			}
			metrics := observability.NewRequestMetrics(time.Now())
			requestDB := svcCtx.DB.Session(&gorm.Session{Logger: observability.NewGormLogger(time.Hour)}).
				WithContext(observability.WithRequestMetrics(context.Background(), metrics))
			requestSvc := *svcCtx
			requestSvc.DB = requestDB
			requestSvc.UserModel = model.NewUserModel(requestDB)
			requestSvc.NotificationModel = model.NewNotificationModel(requestDB)
			service := NewSeasonSettlementService(&requestSvc)
			var notices []seasonRolloverRealtimeNotice
			err := requestDB.Transaction(func(tx *gorm.DB) error {
				var err error
				notices, err = service.persistRolloverNotificationsWithTx(tx, model.Season{Id: 9, Name: "S9"}, nil, records, nil)
				return err
			})
			if err != nil || len(notices) != testCase.users {
				t.Fatalf("persist batched notifications: notices=%d err=%v", len(notices), err)
			}
			if sqlCount := metrics.Snapshot().SQLCount; sqlCount != testCase.wantQueries {
				t.Fatalf("unexpected notification batch query count: got=%d want=%d", sqlCount, testCase.wantQueries)
			}
		})
	}
}

func TestSeasonSettlementRealtimeFailureDoesNotRollbackOrDuplicate(t *testing.T) {
	svcCtx := newSeasonSettlementTestSvc(t)
	season, _, now := seedSeasonSettlementScenario(t, svcCtx, false)
	service := NewSeasonSettlementService(svcCtx)
	notifyCalls := 0
	service.sendRealtime = func(seasonRolloverRealtimeNotice) error {
		notifyCalls++
		settlement, err := svcCtx.SeasonSettlementModel.FindBySeasonId(season.Id)
		storedSeason, seasonErr := svcCtx.SeasonModel.FindById(season.Id)
		if err != nil || settlement == nil || settlement.Status != model.SeasonSettlementStatusCompleted || seasonErr != nil || storedSeason == nil || storedSeason.Status != 2 {
			t.Errorf("realtime delivery must run after final commit: settlement=%+v err=%v season=%+v seasonErr=%v", settlement, err, storedSeason, seasonErr)
		}
		return errors.New("push unavailable")
	}

	if _, err := service.SettleSeasonAt(context.Background(), season.Id, now); err != nil {
		t.Fatalf("settlement must survive realtime failure: %v", err)
	}
	if notifyCalls != 3 {
		t.Fatalf("expected three best-effort realtime attempts, got %d", notifyCalls)
	}
	assertSeasonStatus(t, svcCtx, season.Id, 2)
	assertSeasonSettlementTitles(t, svcCtx, season.Id, 3)
	assertSeasonRolloverNotifications(t, svcCtx, season, nil, 3)

	if _, err := service.SettleSeasonAt(context.Background(), season.Id, now.Add(time.Minute)); err != nil {
		t.Fatalf("retry completed settlement: %v", err)
	}
	if notifyCalls != 3 {
		t.Fatalf("expected completed retry not to resend realtime notifications, got %d calls", notifyCalls)
	}
	assertSeasonSettlementTitles(t, svcCtx, season.Id, 3)
	assertSeasonRolloverNotifications(t, svcCtx, season, nil, 3)
}

func TestSeasonSettlementNotificationPersistenceFailureRollsBackAndRetriesCleanly(t *testing.T) {
	svcCtx := newSeasonSettlementTestSvc(t)
	season, _, now := seedSeasonSettlementScenario(t, svcCtx, false)
	service := NewSeasonSettlementService(svcCtx)
	service.sendRealtime = func(seasonRolloverRealtimeNotice) error { return nil }
	if err := svcCtx.DB.Migrator().DropTable(&model.Notification{}); err != nil {
		t.Fatalf("drop notifications table: %v", err)
	}

	if _, err := service.SettleSeasonAt(context.Background(), season.Id, now); err == nil {
		t.Fatal("expected notification persistence failure")
	}
	assertSeasonStatus(t, svcCtx, season.Id, 1)
	assertSeasonSettlementRecords(t, svcCtx, season.Id)
	assertSeasonSettlementTitles(t, svcCtx, season.Id, 3)
	assertSeasonSettlementSnapshots(t, svcCtx, season.Id, 3)

	failed, err := svcCtx.SeasonSettlementModel.FindBySeasonId(season.Id)
	if err != nil || failed == nil || failed.Status != model.SeasonSettlementStatusFailed || failed.Attempts != 1 {
		t.Fatalf("unexpected failed settlement state: settlement=%+v err=%v", failed, err)
	}
	if failed.RecordsCompletedAt == nil || failed.ChallengesCompletedAt == nil || failed.TitlesCompletedAt == nil || failed.NotificationsCompletedAt != nil {
		t.Fatalf("completed phases must survive notification failure: %+v", failed)
	}
	if err := svcCtx.DB.AutoMigrate(&model.Notification{}); err != nil {
		t.Fatalf("restore notifications table: %v", err)
	}
	if _, err := service.SettleSeasonAt(context.Background(), season.Id, now.Add(time.Minute)); err != nil {
		t.Fatalf("retry settlement after notification recovery: %v", err)
	}
	assertSeasonStatus(t, svcCtx, season.Id, 2)
	assertSeasonSettlementRecords(t, svcCtx, season.Id)
	assertSeasonSettlementTitles(t, svcCtx, season.Id, 3)
	assertSeasonSettlementSnapshots(t, svcCtx, season.Id, 3)
	assertSeasonRolloverNotifications(t, svcCtx, season, nil, 3)
	completed, err := svcCtx.SeasonSettlementModel.FindBySeasonId(season.Id)
	if err != nil || completed == nil || completed.Status != model.SeasonSettlementStatusCompleted || completed.Attempts != 2 {
		t.Fatalf("unexpected recovered settlement state: settlement=%+v err=%v", completed, err)
	}
}

func TestSeasonSettlementResumesCommittedPhasesAndPublishesAtomically(t *testing.T) {
	phases := []string{
		seasonSettlementPhaseRecords,
		seasonSettlementPhaseChallenges,
		seasonSettlementPhaseTitles,
		seasonSettlementPhaseNotifications,
		seasonSettlementPhasePublish,
	}
	for _, interruptedPhase := range phases {
		t.Run(interruptedPhase, func(t *testing.T) {
			svcCtx := newSeasonSettlementTestSvc(t)
			season, next, now := seedSeasonSettlementScenario(t, svcCtx, true)
			service := NewSeasonSettlementService(svcCtx)
			service.sendRealtime = func(seasonRolloverRealtimeNotice) error { return nil }
			interrupted := false
			service.phaseHook = func(phase string) error {
				if phase == interruptedPhase && !interrupted {
					interrupted = true
					return errors.New("simulated interruption after " + phase)
				}
				return nil
			}
			if _, err := service.SettleSeasonAt(context.Background(), season.Id, now); err == nil {
				t.Fatalf("expected interruption after %s", interruptedPhase)
			}
			assertSeasonStatus(t, svcCtx, season.Id, 1)
			assertSeasonStatus(t, svcCtx, next.Id, 0)
			failed, err := svcCtx.SeasonSettlementModel.FindBySeasonId(season.Id)
			if err != nil || failed == nil || failed.Status != model.SeasonSettlementStatusFailed || failed.Attempts != 1 {
				t.Fatalf("unexpected interrupted settlement: %+v err=%v", failed, err)
			}
			wantRecords := true
			wantChallenges := interruptedPhase != seasonSettlementPhaseRecords
			wantTitles := interruptedPhase == seasonSettlementPhaseTitles || interruptedPhase == seasonSettlementPhaseNotifications || interruptedPhase == seasonSettlementPhasePublish
			wantNotifications := interruptedPhase == seasonSettlementPhaseNotifications || interruptedPhase == seasonSettlementPhasePublish
			if (failed.RecordsCompletedAt != nil) != wantRecords ||
				(failed.ChallengesCompletedAt != nil) != wantChallenges ||
				(failed.TitlesCompletedAt != nil) != wantTitles ||
				(failed.NotificationsCompletedAt != nil) != wantNotifications {
				t.Fatalf("unexpected phase checkpoints after %s: %+v", interruptedPhase, failed)
			}

			retry := NewSeasonSettlementService(svcCtx)
			retry.sendRealtime = func(seasonRolloverRealtimeNotice) error { return nil }
			summary, err := retry.SettleSeasonAt(context.Background(), season.Id, now.Add(time.Minute))
			if err != nil || summary.Skipped || summary.SeasonRecords != 3 || summary.SeasonTitles != 3 || summary.ChallengeSnapshots != 3 || summary.Notifications != 3 {
				t.Fatalf("resume settlement after %s: summary=%+v err=%v", interruptedPhase, summary, err)
			}
			completed, err := svcCtx.SeasonSettlementModel.FindBySeasonId(season.Id)
			if err != nil || completed == nil || completed.Status != model.SeasonSettlementStatusCompleted || completed.Attempts != 2 || completed.CompletedAt == nil {
				t.Fatalf("unexpected completed settlement after resume: %+v err=%v", completed, err)
			}
			assertSeasonStatus(t, svcCtx, season.Id, 2)
			assertSeasonStatus(t, svcCtx, next.Id, 1)
			assertSeasonSettlementRecords(t, svcCtx, season.Id)
			assertSeasonSettlementTitles(t, svcCtx, season.Id, 3)
			assertSeasonSettlementSnapshots(t, svcCtx, season.Id, 3)
			assertSeasonRolloverNotifications(t, svcCtx, season, &next, 3)
		})
	}
}

func TestSeasonSettlementServiceConcurrentCallsCreateSingleResult(t *testing.T) {
	svcCtx := newSeasonSettlementTestSvc(t)
	season, _, now := seedSeasonSettlementScenario(t, svcCtx, false)
	services := []*SeasonSettlementService{
		NewSeasonSettlementService(svcCtx),
		NewSeasonSettlementService(svcCtx),
	}
	for _, service := range services {
		service.sendRealtime = func(seasonRolloverRealtimeNotice) error { return nil }
	}

	var wg sync.WaitGroup
	errs := make(chan error, len(services))
	for _, service := range services {
		wg.Add(1)
		go func(service *SeasonSettlementService) {
			defer wg.Done()
			_, err := service.SettleSeasonAt(context.Background(), season.Id, now)
			errs <- err
		}(service)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent settlement failed: %v", err)
		}
	}

	assertSeasonSettlementRecords(t, svcCtx, season.Id)
	assertSeasonSettlementTitles(t, svcCtx, season.Id, 3)
	assertSeasonSettlementSnapshots(t, svcCtx, season.Id, 3)
	assertSeasonRolloverNotifications(t, svcCtx, season, nil, 3)
	var settlementCount int64
	if err := svcCtx.DB.Model(&model.SeasonSettlement{}).Where("season_id = ?", season.Id).Count(&settlementCount).Error; err != nil {
		t.Fatalf("count settlements: %v", err)
	}
	if settlementCount != 1 {
		t.Fatalf("expected one settlement row, got %d", settlementCount)
	}
}

func newSeasonSettlementTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&_busy_timeout=5000"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sqlite db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(
		&model.User{},
		&model.Achievement{},
		&model.UserAchievement{},
		&model.Season{},
		&model.SeasonRecord{},
		&model.SeasonSettlement{},
		&model.SeasonChallengeSnapshot{},
		&model.AchievementProgressEvent{},
		&model.Match{},
		&model.MatchRound{},
		&model.RankChangeLog{},
		&model.UserTitle{},
		&model.Notification{},
	); err != nil {
		t.Fatalf("prepare season settlement schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:                            db,
		UserModel:                     model.NewUserModel(db),
		MatchModel:                    model.NewMatchModel(db),
		RankingModel:                  model.NewRankingModel(db),
		UserTitleModel:                model.NewUserTitleModel(db),
		AchievementProgressEventModel: model.NewAchievementProgressEventModel(db),
		NotificationModel:             model.NewNotificationModel(db),
		SeasonModel:                   model.NewSeasonModel(db),
		SeasonRecordModel:             model.NewSeasonRecordModel(db),
		SeasonChallengeSnapshotModel:  model.NewSeasonChallengeSnapshotModel(db),
		SeasonSettlementModel:         model.NewSeasonSettlementModel(db),
	}
}

func seedSeasonSettlementScenario(t *testing.T, svcCtx *svc.ServiceContext, withNext bool) (model.Season, model.Season, time.Time) {
	t.Helper()
	season := model.Season{
		Id:        3,
		Name:      "S3",
		StartDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
		Status:    1,
	}
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	if err := svcCtx.SeasonModel.Create(&season); err != nil {
		t.Fatalf("create current season: %v", err)
	}
	next := model.Season{}
	if withNext {
		next = model.Season{
			Id:        4,
			Name:      "S4",
			StartDate: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
			Status:    0,
		}
		if err := svcCtx.SeasonModel.Create(&next); err != nil {
			t.Fatalf("create next season: %v", err)
		}
	}

	for _, userID := range []int64{10, 20, 30} {
		if err := svcCtx.UserModel.Create(&model.User{Id: userID, Nickname: "用户"}); err != nil {
			t.Fatalf("create user %d: %v", userID, err)
		}
	}
	seedSeasonSettlementMatch(t, svcCtx, 301, 10, 30, 1, time.Date(2026, 7, 10, 20, 0, 0, 0, time.UTC))
	seedSeasonSettlementMatch(t, svcCtx, 302, 20, 30, 1, time.Date(2026, 7, 20, 20, 0, 0, 0, time.UTC))
	seedSeasonRankLog(t, svcCtx, 10, 301, 1400, 1500, time.Date(2026, 7, 10, 20, 0, 0, 0, time.UTC))
	seedSeasonRankLog(t, svcCtx, 30, 301, 1500, 1450, time.Date(2026, 7, 10, 20, 0, 0, 0, time.UTC))
	seedSeasonRankLog(t, svcCtx, 20, 302, 1400, 1500, time.Date(2026, 7, 20, 20, 0, 0, 0, time.UTC))
	seedSeasonRankLog(t, svcCtx, 30, 302, 1450, 1400, time.Date(2026, 7, 20, 20, 0, 0, 0, time.UTC))
	if err := svcCtx.SeasonRecordModel.CreateBatch([]model.SeasonRecord{
		{Id: 3, SeasonId: season.Id, UserId: 10, GameType: 3, StartRankScore: 1400, EndRankScore: 1500, PeakRankScore: 1500, MatchesPlayed: 1, Wins: 1},
		{Id: 2, SeasonId: season.Id, UserId: 20, GameType: 3, StartRankScore: 1400, EndRankScore: 1500, PeakRankScore: 1500, MatchesPlayed: 1, Wins: 1},
		{Id: 1, SeasonId: season.Id, UserId: 30, GameType: 3, StartRankScore: 1500, EndRankScore: 1400, PeakRankScore: 1500, MatchesPlayed: 2, Wins: 0},
	}); err != nil {
		t.Fatalf("seed incremental season records: %v", err)
	}

	events := []*model.AchievementProgressEvent{
		model.NewAchievementProgressEvent(10, achievementx.SourceTypeMatch, 301, 3, achievementx.MetricMatchesTotal, 20, time.Date(2026, 7, 10, 20, 0, 0, 0, time.UTC)),
		model.NewAchievementProgressEvent(10, achievementx.SourceTypeMatch, 302, 3, achievementx.MetricWinsTotal, 10, time.Date(2026, 7, 20, 20, 0, 0, 0, time.UTC)),
		model.NewAchievementProgressEvent(10, achievementx.SourceTypeTournamentFinish, 401, 3, achievementx.MetricTournamentFinishTotal, 1, time.Date(2026, 7, 25, 20, 0, 0, 0, time.UTC)),
	}
	for _, event := range events {
		if _, err := svcCtx.AchievementProgressEventModel.CreateIfAbsent(event); err != nil {
			t.Fatalf("create challenge event: %v", err)
		}
	}
	return season, next, now
}

func seedSeasonSettlementMatch(t *testing.T, svcCtx *svc.ServiceContext, matchID, userID, opponentID int64, result int, endedAt time.Time) {
	t.Helper()
	match := model.Match{
		Id:            matchID,
		UserId:        userID,
		OpponentId:    &opponentID,
		OpponentName:  "对手",
		GameType:      3,
		MatchMode:     model.MatchModeRanked,
		MyScore:       2,
		OpponentScore: 1,
		Status:        2,
		Result:        &result,
		MatchTime:     endedAt.Add(-time.Hour),
		EndTime:       &endedAt,
	}
	if err := svcCtx.MatchModel.Create(&match); err != nil {
		t.Fatalf("create season match %d: %v", matchID, err)
	}
	winner := 1
	if err := svcCtx.DB.Create(&model.MatchRound{MatchId: matchID, RoundNo: 1, Winner: &winner, WinType: "normal"}).Error; err != nil {
		t.Fatalf("create season match round: %v", err)
	}
}

func seedSeasonRankLog(t *testing.T, svcCtx *svc.ServiceContext, userID, matchID int64, before, after int, effectiveAt time.Time) {
	t.Helper()
	result := "lose"
	if after > before {
		result = "win"
	}
	if err := svcCtx.DB.Create(&model.RankChangeLog{
		UserId:      userID,
		MatchId:     matchID,
		ChangeType:  "match_result",
		GameType:    3,
		Result:      result,
		FinalChange: after - before,
		BeforeScore: before,
		AfterScore:  after,
		BeforeLevel: 1,
		AfterLevel:  1,
		Remark:      "{}",
		EffectiveAt: effectiveAt,
	}).Error; err != nil {
		t.Fatalf("create rank log user=%d match=%d: %v", userID, matchID, err)
	}
}

func assertCareerAssetsPreserved(t *testing.T, svcCtx *svc.ServiceContext, userID, achievementID int64, unlockedAt, equippedAt time.Time) {
	t.Helper()
	var userAchievement model.UserAchievement
	if err := svcCtx.DB.Where("user_id = ? AND achievement_id = ?", userID, achievementID).First(&userAchievement).Error; err != nil {
		t.Fatalf("find preserved career achievement: %v", err)
	}
	if userAchievement.Progress != 100 || userAchievement.Unlocked != 1 || userAchievement.UnlockedAt == nil || !userAchievement.UnlockedAt.Equal(unlockedAt) {
		t.Fatalf("career achievement changed during settlement: %+v", userAchievement)
	}
	var title model.UserTitle
	if err := svcCtx.DB.Where("user_id = ? AND source_type = ? AND source_ref_id = ?", userID, achievementx.SourceTypeAchievement, achievementID).First(&title).Error; err != nil {
		t.Fatalf("find preserved career title: %v", err)
	}
	if title.TitleName != "资深球手" || title.Equipped != 1 || title.EquippedAt == nil || !title.EquippedAt.Equal(equippedAt) {
		t.Fatalf("career title changed during settlement: %+v", title)
	}
}

func assertSeasonStatus(t *testing.T, svcCtx *svc.ServiceContext, seasonID int64, want int) {
	t.Helper()
	season, err := svcCtx.SeasonModel.FindById(seasonID)
	if err != nil || season == nil {
		t.Fatalf("find season %d: season=%+v err=%v", seasonID, season, err)
	}
	if season.Status != want {
		t.Fatalf("expected season %d status %d, got %d", seasonID, want, season.Status)
	}
}

func assertSingleActiveSeason(t *testing.T, svcCtx *svc.ServiceContext, wantSeasonID int64) {
	t.Helper()
	var active []model.Season
	if err := svcCtx.DB.Where("status = ?", 1).Order("id ASC").Find(&active).Error; err != nil {
		t.Fatalf("find active seasons: %v", err)
	}
	if wantSeasonID == 0 {
		if len(active) != 0 {
			t.Fatalf("expected intermission, got active seasons %+v", active)
		}
		return
	}
	if len(active) != 1 || active[0].Id != wantSeasonID {
		t.Fatalf("expected only active season %d, got %+v", wantSeasonID, active)
	}
}

func assertSeasonSettlementRecords(t *testing.T, svcCtx *svc.ServiceContext, seasonID int64) {
	t.Helper()
	var records []model.SeasonRecord
	if err := svcCtx.DB.Where("season_id = ?", seasonID).Order("final_rank ASC").Find(&records).Error; err != nil {
		t.Fatalf("find season records: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected three season records, got %+v", records)
	}
	wantUsers := []int64{10, 20, 30}
	wantMatches := []int{1, 1, 2}
	wantWins := []int{1, 1, 0}
	for i, record := range records {
		if record.UserId != wantUsers[i] || record.FinalRank != i+1 || record.MatchesPlayed != wantMatches[i] || record.Wins != wantWins[i] {
			t.Fatalf("unexpected record %d: %+v", i, record)
		}
	}
}

func assertSeasonSettlementTitles(t *testing.T, svcCtx *svc.ServiceContext, seasonID int64, want int64) {
	t.Helper()
	var titles []model.UserTitle
	if err := svcCtx.DB.Where("source_type = ? AND source_ref_id = ?", achievementx.SourceTypeSeason, seasonID).Order("user_id ASC").Find(&titles).Error; err != nil {
		t.Fatalf("find season titles: %v", err)
	}
	if int64(len(titles)) != want {
		t.Fatalf("expected %d season titles, got %+v", want, titles)
	}
	wantNames := []string{"S3中式八球赛季冠军", "S3中式八球赛季亚军", "S3中式八球赛季季军"}
	for i, title := range titles {
		if title.TitleName != wantNames[i] {
			t.Fatalf("unexpected title %d: %+v", i, title)
		}
	}
}

func assertSeasonSettlementSnapshots(t *testing.T, svcCtx *svc.ServiceContext, seasonID int64, want int64) {
	t.Helper()
	var snapshots []model.SeasonChallengeSnapshot
	if err := svcCtx.DB.Where("season_id = ?", seasonID).Order("id ASC").Find(&snapshots).Error; err != nil {
		t.Fatalf("find challenge snapshots: %v", err)
	}
	if int64(len(snapshots)) != want {
		t.Fatalf("expected %d snapshots, got %+v", want, snapshots)
	}
	for _, snapshot := range snapshots {
		if snapshot.UserId != 10 || snapshot.Completed != 1 {
			t.Fatalf("unexpected challenge snapshot: %+v", snapshot)
		}
	}
}

func assertSeasonRolloverNotifications(t *testing.T, svcCtx *svc.ServiceContext, from model.Season, to *model.Season, want int64) {
	t.Helper()
	var notifications []model.Notification
	if err := svcCtx.DB.Where("type = ?", "season_rollover").Order("user_id ASC").Find(&notifications).Error; err != nil {
		t.Fatalf("find season rollover notifications: %v", err)
	}
	if int64(len(notifications)) != want {
		t.Fatalf("expected %d rollover notifications, got %+v", want, notifications)
	}
	toID := int64(0)
	if to != nil {
		toID = to.Id
	}
	wantDedupe := seasonRolloverDedupeKey(from.Id, toID)
	for _, notification := range notifications {
		if notification.DedupeKey == nil || *notification.DedupeKey != wantDedupe {
			t.Fatalf("unexpected notification dedupe key: %+v", notification.DedupeKey)
		}
		if notification.Data == nil {
			t.Fatalf("expected notification data: %+v", notification)
		}
		var payload seasonRolloverNotificationData
		if err := json.Unmarshal([]byte(*notification.Data), &payload); err != nil {
			t.Fatalf("decode rollover payload: %v", err)
		}
		if payload.FromSeasonId != from.Id || payload.ToSeasonId != toID || payload.CareerAchievementsPreserved != true {
			t.Fatalf("unexpected rollover payload: %+v", payload)
		}
		if payload.Intermission != (to == nil) {
			t.Fatalf("unexpected intermission flag: %+v", payload)
		}
	}
}
