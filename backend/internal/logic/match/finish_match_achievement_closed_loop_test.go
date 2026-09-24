package match

import (
	"context"
	"testing"
	"time"

	achievementx "chasing_points/internal/logic/achievement"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newMatchAchievementClosedLoopTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Match{},
		&model.MatchRound{},
		&model.MatchAction{},
		&model.MatchAchievement{},
		&model.UserRanking{},
		&model.RankChangeLog{},
		&model.Notification{},
		&model.AchievementRewardConfig{},
		&model.Achievement{},
		&model.UserAchievement{},
		&model.UserTitle{},
		&model.AchievementProgressEvent{},
	); err != nil {
		t.Fatalf("prepare match achievement closed loop schema: %v", err)
	}
	if err := db.Exec("DROP TABLE IF EXISTS user_ranking").Error; err != nil {
		t.Fatalf("drop sqlite user_ranking table: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE user_ranking (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			game_type INTEGER NOT NULL DEFAULT 3,
			rank_score INTEGER NOT NULL DEFAULT 0,
			rank_level INTEGER NOT NULL DEFAULT 1,
			total_wins INTEGER NOT NULL DEFAULT 0,
			total_losses INTEGER NOT NULL DEFAULT 0,
			current_streak INTEGER NOT NULL DEFAULT 0,
			max_streak INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			UNIQUE(user_id, game_type)
		)
	`).Error; err != nil {
		t.Fatalf("recreate sqlite user_ranking table: %v", err)
	}

	return &svc.ServiceContext{
		DB:                            db,
		UserModel:                     model.NewUserModel(db),
		MatchModel:                    model.NewMatchModel(db),
		RankingModel:                  model.NewRankingModel(db),
		NotificationModel:             model.NewNotificationModel(db),
		AchievementModel:              model.NewAchievementModel(db),
		UserAchievementModel:          model.NewUserAchievementModel(db),
		UserTitleModel:                model.NewUserTitleModel(db),
		AchievementProgressEventModel: model.NewAchievementProgressEventModel(db),
	}
}

func matchAchievementCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func TestAppendMatchAchievementProgressEventUsesMatchEffectiveTime(t *testing.T) {
	svcCtx := newMatchAchievementClosedLoopTestSvc(t)
	service := achievementx.NewAchievementProgressService(svcCtx)
	endedAt := time.Date(2026, 7, 22, 20, 30, 0, 0, time.UTC)
	match := &model.Match{Id: 6999, GameType: 3, EndTime: &endedAt}

	if err := appendMatchAchievementProgressEvent(service, 1001, match, achievementx.MetricMatchesTotal, 1); err != nil {
		t.Fatalf("append match progress event: %v", err)
	}
	event, err := svcCtx.AchievementProgressEventModel.FindBySourceMetric(1001, achievementx.SourceTypeMatch, match.Id, achievementx.MetricMatchesTotal)
	if err != nil || event == nil {
		t.Fatalf("find progress event: event=%+v err=%v", event, err)
	}
	if !event.OccurredAt.Equal(endedAt) {
		t.Fatalf("expected occurred_at %v, got %v", endedAt, event.OccurredAt)
	}
}

func TestEndRoundStoresSpecialRecordForWinningActor(t *testing.T) {
	svcCtx := newMatchAchievementClosedLoopTestSvc(t)
	opponentID := int64(2002)
	seedMatchAchievementUsers(t, svcCtx, 1001, opponentID)
	seedMatchAchievementMatch(t, svcCtx, &model.Match{
		Id:           7001,
		UserId:       1001,
		OpponentId:   &opponentID,
		OpponentName: "选手乙",
		GameType:     3,
		Status:       1,
		MatchTime:    time.Now(),
		SyncRevision: 0,
	})

	resp, err := NewEndRoundLogic(matchAchievementCtx(1001), svcCtx).EndRound(&types.EndRoundReq{
		MatchId:        7001,
		Winner:         2,
		Score:          1,
		WinType:        "break_clear",
		ClientActionId: "round-achievement-7001",
		BaseRevision:   0,
	})
	if err != nil {
		t.Fatalf("end round: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected end round success, got %#v", resp)
	}

	var stored model.MatchAchievement
	if err := svcCtx.DB.Where("match_id = ? AND actor = ? AND achievement_type = ?", 7001, 2, "break_and_run").
		First(&stored).Error; err != nil {
		t.Fatalf("find actor 2 special record: %v", err)
	}
	if stored.Count != 1 {
		t.Fatalf("expected actor 2 special record count 1, got %d", stored.Count)
	}
}

func TestFinishMatchWritesAchievementProgressAndIsIdempotent(t *testing.T) {
	svcCtx := newMatchAchievementClosedLoopTestSvc(t)
	opponentID := int64(2002)
	seedMatchAchievementUsers(t, svcCtx, 1001, opponentID)
	seedMatchAchievementDefinitions(t, svcCtx)
	seedMatchAchievementMatch(t, svcCtx, &model.Match{
		Id:            7002,
		UserId:        1001,
		OpponentId:    &opponentID,
		OpponentName:  "选手乙",
		GameType:      3,
		MyScore:       2,
		OpponentScore: 1,
		Status:        1,
		MatchTime:     time.Now().Add(-20 * time.Minute),
		SyncRevision:  0,
	})
	seedMatchAchievementRound(t, svcCtx, 7002, 1, 1)
	if err := svcCtx.MatchModel.SaveAchievementWithTx(nil, 7002, "break_and_run", 1, 2); err != nil {
		t.Fatalf("seed actor 2 special record: %v", err)
	}

	resp, err := NewFinishMatchLogic(matchAchievementCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId:        7002,
		ClientActionId: "finish-achievement-7002",
		BaseRevision:   0,
	})
	if err != nil {
		t.Fatalf("finish match: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected finish match success, got %#v", resp)
	}

	assertUserAchievementProgress(t, svcCtx, 1001, "match_1", 1, true)
	assertUserAchievementProgress(t, svcCtx, 2002, "match_1", 1, true)
	assertUserAchievementProgress(t, svcCtx, 1001, "wins_1", 1, true)
	assertUserAchievementProgress(t, svcCtx, 1001, "streak_1", 1, true)
	assertUserAchievementProgress(t, svcCtx, 2002, "break_clear_1", 1, true)
	assertUserAchievementProgress(t, svcCtx, 1001, "break_clear_1", 0, false)
	firstSyncedAt := assertMatchAchievementSynced(t, svcCtx, 7002, true)

	replayResp, err := NewFinishMatchLogic(matchAchievementCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId:        7002,
		ClientActionId: "finish-achievement-7002",
		BaseRevision:   0,
	})
	if err != nil {
		t.Fatalf("replay finish match: %v", err)
	}
	if !replayResp.Success {
		t.Fatalf("expected replay finish success, got %#v", replayResp)
	}

	if countProgressEvents(t, svcCtx, 1001, achievementx.MetricMatchesTotal) != 1 {
		t.Fatal("expected player1 matches_total to stay idempotent")
	}
	if countProgressEvents(t, svcCtx, 2002, achievementx.MetricBreakClearTotal) != 1 {
		t.Fatal("expected player2 break_clear_total to stay idempotent")
	}
	assertUserAchievementProgress(t, svcCtx, 1001, "match_1", 1, true)
	assertUserAchievementProgress(t, svcCtx, 2002, "break_clear_1", 1, true)
	replayedSyncedAt := assertMatchAchievementSynced(t, svcCtx, 7002, true)
	if !replayedSyncedAt.Equal(*firstSyncedAt) {
		t.Fatalf("expected replay to preserve synced_at %v, got %v", firstSyncedAt, replayedSyncedAt)
	}
}

func TestFinishPracticeMatchMarksAchievementSyncReadyWithoutRewards(t *testing.T) {
	svcCtx := newMatchAchievementClosedLoopTestSvc(t)
	opponentID := int64(2002)
	seedMatchAchievementUsers(t, svcCtx, 1001, opponentID)
	seedMatchAchievementDefinitions(t, svcCtx)
	seedMatchAchievementMatch(t, svcCtx, &model.Match{
		Id:            7004,
		UserId:        1001,
		OpponentId:    &opponentID,
		OpponentName:  "选手乙",
		GameType:      3,
		MatchMode:     model.MatchModePractice,
		MyScore:       1,
		OpponentScore: 0,
		Status:        1,
		MatchTime:     time.Now().Add(-10 * time.Minute),
	})
	seedMatchAchievementRound(t, svcCtx, 7004, 1, 1)

	resp, err := NewFinishMatchLogic(matchAchievementCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId:        7004,
		ClientActionId: "finish-practice-7004",
		BaseRevision:   0,
	})
	if err != nil || !resp.Success {
		t.Fatalf("finish practice match: resp=%+v err=%v", resp, err)
	}
	assertMatchAchievementSynced(t, svcCtx, 7004, true)
	if total := countAllProgressEvents(t, svcCtx); total != 0 {
		t.Fatalf("expected practice match to create no progress events, got %d", total)
	}
}

func TestFinishDrawMarksAchievementSyncReadyWithoutRewards(t *testing.T) {
	svcCtx := newMatchAchievementClosedLoopTestSvc(t)
	opponentID := int64(2002)
	seedMatchAchievementUsers(t, svcCtx, 1001, opponentID)
	seedMatchAchievementDefinitions(t, svcCtx)
	seedMatchAchievementMatch(t, svcCtx, &model.Match{
		Id:            7005,
		UserId:        1001,
		OpponentId:    &opponentID,
		OpponentName:  "选手乙",
		GameType:      3,
		MatchMode:     model.MatchModeRanked,
		MyScore:       1,
		OpponentScore: 1,
		Status:        1,
		MatchTime:     time.Now().Add(-10 * time.Minute),
	})
	seedMatchAchievementRound(t, svcCtx, 7005, 1, 1)

	resp, err := NewFinishMatchLogic(matchAchievementCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId:        7005,
		ClientActionId: "finish-draw-7005",
		BaseRevision:   0,
	})
	if err != nil || !resp.Success {
		t.Fatalf("finish draw match: resp=%+v err=%v", resp, err)
	}
	assertMatchAchievementSynced(t, svcCtx, 7005, true)
	if total := countAllProgressEvents(t, svcCtx); total != 0 {
		t.Fatalf("expected draw to create no progress events, got %d", total)
	}
}

func TestFinishMatchLeavesAchievementSyncPendingOnFailure(t *testing.T) {
	svcCtx := newMatchAchievementClosedLoopTestSvc(t)
	opponentID := int64(2002)
	seedMatchAchievementUsers(t, svcCtx, 1001, opponentID)
	seedMatchAchievementDefinitions(t, svcCtx)
	seedMatchAchievementMatch(t, svcCtx, &model.Match{
		Id:            7006,
		UserId:        1001,
		OpponentId:    &opponentID,
		OpponentName:  "选手乙",
		GameType:      3,
		MatchMode:     model.MatchModeRanked,
		MyScore:       1,
		OpponentScore: 0,
		Status:        1,
		MatchTime:     time.Now().Add(-10 * time.Minute),
	})
	seedMatchAchievementRound(t, svcCtx, 7006, 1, 1)
	svcCtx.AchievementProgressEventModel = nil

	resp, err := NewFinishMatchLogic(matchAchievementCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId:        7006,
		ClientActionId: "finish-sync-failure-7006",
		BaseRevision:   0,
	})
	if err != nil || !resp.Success {
		t.Fatalf("finish match with achievement sync failure: resp=%+v err=%v", resp, err)
	}
	assertMatchAchievementSynced(t, svcCtx, 7006, false)
}

func TestFinishMatchSkipsAchievementProgressForInvalidMatch(t *testing.T) {
	svcCtx := newMatchAchievementClosedLoopTestSvc(t)
	seedMatchAchievementUsers(t, svcCtx, 1001)
	seedMatchAchievementDefinitions(t, svcCtx)
	seedMatchAchievementMatch(t, svcCtx, &model.Match{
		Id:            7003,
		UserId:        1001,
		OpponentName:  "游客",
		GameType:      3,
		MyScore:       1,
		OpponentScore: 0,
		Status:        1,
		MatchTime:     time.Now().Add(-10 * time.Minute),
		SyncRevision:  0,
	})
	seedMatchAchievementRound(t, svcCtx, 7003, 1, 1)

	resp, err := NewFinishMatchLogic(matchAchievementCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId:        7003,
		ClientActionId: "finish-achievement-7003",
		BaseRevision:   0,
	})
	if err != nil {
		t.Fatalf("finish match: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected finish match success, got %#v", resp)
	}

	var total int64
	if err := svcCtx.DB.Model(&model.AchievementProgressEvent{}).Count(&total).Error; err != nil {
		t.Fatalf("count progress events: %v", err)
	}
	if total != 0 {
		t.Fatalf("expected invalid match to skip progress events, got %d", total)
	}
}

func seedMatchAchievementUsers(t *testing.T, svcCtx *svc.ServiceContext, userIDs ...int64) {
	t.Helper()
	for _, userID := range userIDs {
		user := model.User{Id: userID, Nickname: "测试用户"}
		if err := svcCtx.UserModel.Create(&user); err != nil {
			t.Fatalf("seed user %d: %v", userID, err)
		}
	}
}

func seedMatchAchievementDefinitions(t *testing.T, svcCtx *svc.ServiceContext) {
	t.Helper()
	defs := []model.Achievement{
		{Key: "match_1", Name: "首场对局", Category: "match", GameType: 0, MetricKey: achievementx.MetricMatchesTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 1, Status: 1},
		{Key: "wins_1", Name: "首胜", Category: "wins", GameType: 0, MetricKey: achievementx.MetricWinsTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 1, Status: 1},
		{Key: "streak_1", Name: "一连胜", Category: "streak", GameType: 0, MetricKey: achievementx.MetricMaxWinStreak, ProgressMode: achievementx.ProgressModeMax, Threshold: 1, Status: 1},
		{Key: "break_clear_1", Name: "首次清台", Category: "special", GameType: 0, MetricKey: achievementx.MetricBreakClearTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 1, Status: 1},
	}
	for i := range defs {
		if err := svcCtx.DB.Create(&defs[i]).Error; err != nil {
			t.Fatalf("seed achievement %s: %v", defs[i].Key, err)
		}
	}
}

func seedMatchAchievementMatch(t *testing.T, svcCtx *svc.ServiceContext, match *model.Match) {
	t.Helper()
	if err := svcCtx.MatchModel.Create(match); err != nil {
		t.Fatalf("seed match %d: %v", match.Id, err)
	}
}

func seedMatchAchievementRound(t *testing.T, svcCtx *svc.ServiceContext, matchID int64, roundNo int, winner int) {
	t.Helper()
	if err := svcCtx.DB.Create(&model.MatchRound{
		MatchId:       matchID,
		RoundNo:       roundNo,
		MyScore:       1,
		OpponentScore: 0,
		Winner:        &winner,
		WinType:       "normal",
	}).Error; err != nil {
		t.Fatalf("seed round: %v", err)
	}
}

func assertUserAchievementProgress(t *testing.T, svcCtx *svc.ServiceContext, userID int64, key string, wantProgress int, wantUnlocked bool) {
	t.Helper()

	var achievement model.Achievement
	if err := svcCtx.DB.Where("`key` = ?", key).First(&achievement).Error; err != nil {
		t.Fatalf("find achievement %s: %v", key, err)
	}
	var ua model.UserAchievement
	err := svcCtx.DB.Where("user_id = ? AND achievement_id = ?", userID, achievement.Id).First(&ua).Error
	if err != nil {
		t.Fatalf("find user achievement user=%d key=%s: %v", userID, key, err)
	}
	if ua.Progress != wantProgress {
		t.Fatalf("expected user=%d key=%s progress %d, got %d", userID, key, wantProgress, ua.Progress)
	}
	gotUnlocked := ua.Unlocked == 1
	if gotUnlocked != wantUnlocked {
		t.Fatalf("expected user=%d key=%s unlocked=%v, got %v", userID, key, wantUnlocked, gotUnlocked)
	}
}

func countProgressEvents(t *testing.T, svcCtx *svc.ServiceContext, userID int64, metricKey string) int64 {
	t.Helper()

	var total int64
	if err := svcCtx.DB.Model(&model.AchievementProgressEvent{}).
		Where("user_id = ? AND metric_key = ?", userID, metricKey).
		Count(&total).Error; err != nil {
		t.Fatalf("count progress events: %v", err)
	}
	return total
}

func countAllProgressEvents(t *testing.T, svcCtx *svc.ServiceContext) int64 {
	t.Helper()

	var total int64
	if err := svcCtx.DB.Model(&model.AchievementProgressEvent{}).Count(&total).Error; err != nil {
		t.Fatalf("count all progress events: %v", err)
	}
	return total
}

func assertMatchAchievementSynced(t *testing.T, svcCtx *svc.ServiceContext, matchID int64, wantSynced bool) *time.Time {
	t.Helper()

	match, err := svcCtx.MatchModel.FindById(matchID)
	if err != nil || match == nil {
		t.Fatalf("find match %d: match=%+v err=%v", matchID, match, err)
	}
	gotSynced := match.AchievementSyncedAt != nil
	if gotSynced != wantSynced {
		t.Fatalf("expected match %d achievement_synced_at present=%v, got %v", matchID, wantSynced, match.AchievementSyncedAt)
	}
	return match.AchievementSyncedAt
}
