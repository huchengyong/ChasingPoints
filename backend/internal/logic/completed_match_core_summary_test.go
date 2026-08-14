package logic

import (
	"context"
	"fmt"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	"chasing_points/internal/svc"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCompletedMatchCoreSummaryUsesFixedMatchScopedQueries(t *testing.T) {
	one := completedMatchCoreQueryCount(t, 1)
	hundred := completedMatchCoreQueryCount(t, 100)
	if one != 4 || hundred != 4 {
		t.Fatalf("completed match core must use four fixed match-scoped queries: one=%d hundred=%d", one, hundred)
	}
}

func completedMatchCoreQueryCount(t *testing.T, rows int) int64 {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+fmt.Sprintf("-%d?mode=memory&cache=shared", rows)), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}, &model.MatchRound{}, &model.MatchAction{}, &model.MatchAchievement{}); err != nil {
		t.Fatalf("prepare core query schema: %v", err)
	}
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "甲"}, {Id: 2, Nickname: "乙"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	result := 1
	opponentID := int64(2)
	completedAt := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	target := model.Match{Id: 1, UserId: 1, OpponentId: &opponentID, OpponentName: "乙", GameType: 3, Status: 2, Result: &result, MatchTime: completedAt.Add(-time.Minute), CompletedAt: &completedAt}
	if err := db.Create(&target).Error; err != nil {
		t.Fatalf("seed target match: %v", err)
	}
	winner := 1
	if err := db.Create(&model.MatchRound{MatchId: 1, RoundNo: 1, Winner: &winner}).Error; err != nil {
		t.Fatalf("seed target round: %v", err)
	}
	if err := db.Create(&model.MatchAction{MatchId: 1, RoundNo: 1, ActionType: "score", Actor: 1}).Error; err != nil {
		t.Fatalf("seed target action: %v", err)
	}
	if err := db.Create(&model.MatchAchievement{MatchId: 1, Actor: 1, AchievementType: "break_50", Count: 1}).Error; err != nil {
		t.Fatalf("seed target achievement: %v", err)
	}
	matches := make([]model.Match, 0, rows)
	rounds := make([]model.MatchRound, 0, rows)
	actions := make([]model.MatchAction, 0, rows)
	achievements := make([]model.MatchAchievement, 0, rows)
	for index := 1; index <= rows; index++ {
		matchID := int64(1000 + index)
		matches = append(matches, model.Match{Id: matchID, UserId: 1, OpponentId: &opponentID, GameType: 3, Status: 2, Result: &result, MatchTime: completedAt})
		rounds = append(rounds, model.MatchRound{MatchId: matchID, RoundNo: 1, Winner: &winner})
		actions = append(actions, model.MatchAction{MatchId: matchID, RoundNo: 1, ActionType: "score", Actor: 1})
		achievements = append(achievements, model.MatchAchievement{MatchId: matchID, Actor: 1, AchievementType: "break_50", Count: 1})
	}
	if err := db.Create(&matches).Error; err != nil {
		t.Fatalf("seed unrelated matches: %v", err)
	}
	if err := db.Create(&rounds).Error; err != nil {
		t.Fatalf("seed unrelated rounds: %v", err)
	}
	if err := db.Create(&actions).Error; err != nil {
		t.Fatalf("seed unrelated actions: %v", err)
	}
	if err := db.Create(&achievements).Error; err != nil {
		t.Fatalf("seed unrelated achievements: %v", err)
	}
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.Background(), metrics)
	requestDB := db.WithContext(ctx)
	core, err := BuildCompletedMatchCoreSummary(ctx, &svc.ServiceContext{UserModel: model.NewUserModel(requestDB), MatchModel: model.NewMatchModel(requestDB)}, &target)
	if err != nil || core == nil || core.Match.Id != 1 {
		t.Fatalf("build completed match core: core=%+v err=%v", core, err)
	}
	return metrics.Snapshot().SQLCount
}

func TestReadyCompletedMatchCoreCacheKeepsCurrentProfilesOutsideLongTTL(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}, &model.MatchRound{}, &model.MatchAction{}, &model.MatchAchievement{}, &model.RankChangeLog{}); err != nil {
		t.Fatalf("prepare ready core schema: %v", err)
	}
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "旧昵称"}, {Id: 2, Nickname: "乙"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	result := 1
	opponentID := int64(2)
	completedAt := time.Now()
	match := model.Match{Id: 20, UserId: 1, OpponentId: &opponentID, OpponentName: "乙", GameType: 3, Status: 2, Result: &result, MyScore: 5, OpponentScore: 2, MatchTime: completedAt.Add(-time.Minute), CompletedAt: &completedAt, AchievementSyncedAt: &completedAt}
	if err := db.Create(&match).Error; err != nil {
		t.Fatalf("seed ready match: %v", err)
	}
	winner := 1
	if err := db.Create(&model.MatchRound{MatchId: match.Id, RoundNo: 1, Winner: &winner}).Error; err != nil {
		t.Fatalf("seed round: %v", err)
	}
	miniRedis := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	ResetCompletedMatchCoreCacheForTest()

	load := func() (*CompletedMatchCoreSummary, int64) {
		metrics := observability.NewRequestMetrics(time.Now())
		ctx := observability.WithRequestMetrics(context.Background(), metrics)
		requestDB := db.WithContext(ctx)
		core, err := BuildCompletedMatchCoreSummary(ctx, &svc.ServiceContext{
			Redis: redisClient, UserModel: model.NewUserModel(requestDB), MatchModel: model.NewMatchModel(requestDB), RankingModel: model.NewRankingModel(requestDB),
		}, &match)
		if err != nil {
			t.Fatalf("build ready core: %v", err)
		}
		return core, metrics.Snapshot().SQLCount
	}
	first, firstQueries := load()
	if err := db.Model(&model.User{}).Where("id = ?", 1).Update("nickname", "新昵称").Error; err != nil {
		t.Fatalf("update current profile: %v", err)
	}
	second, secondQueries := load()
	if firstQueries != 6 || secondQueries != 1 {
		t.Fatalf("ready core cache must retain only immutable facts: first=%d second=%d", firstQueries, secondQueries)
	}
	if first.Player1 == nil || first.Player1.Nickname != "旧昵称" || second.Player1 == nil || second.Player1.Nickname != "新昵称" {
		t.Fatalf("current profiles must be reloaded outside core cache: first=%+v second=%+v", first.Player1, second.Player1)
	}
	metrics := CompletedMatchCoreCacheMetrics()
	if metrics.Hits != 1 || metrics.Misses != 1 || metrics.Fallbacks != 1 {
		t.Fatalf("unexpected completed core cache metrics: %+v", metrics)
	}
}

func TestPendingCompletedMatchCoreIsExcludedFromLongTTLCache(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Match{}, &model.MatchRound{}, &model.MatchAction{}, &model.MatchAchievement{}); err != nil {
		t.Fatalf("prepare pending core schema: %v", err)
	}
	result := 1
	match := model.Match{Id: 21, UserId: 1, GameType: 3, Status: 2, Result: &result, MatchTime: time.Now()}
	miniRedis := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	ResetCompletedMatchCoreCacheForTest()
	svcCtx := &svc.ServiceContext{Redis: redisClient, MatchModel: model.NewMatchModel(db)}
	if _, err := BuildCompletedMatchCoreSummary(context.Background(), svcCtx, &match); err != nil {
		t.Fatalf("build pending core: %v", err)
	}
	if len(miniRedis.Keys()) != 0 || CompletedMatchCoreCacheMetrics().Hits != 0 || CompletedMatchCoreCacheMetrics().Misses != 0 {
		t.Fatalf("pending core must not enter long cache: keys=%v metrics=%+v", miniRedis.Keys(), CompletedMatchCoreCacheMetrics())
	}
}

func TestCompletedMatchCoreSummaryOnlyLoadsTargetMatchFacts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}, &model.MatchRound{}, &model.MatchAction{}, &model.MatchAchievement{}); err != nil {
		t.Fatalf("prepare core schema: %v", err)
	}
	svcCtx := &svc.ServiceContext{
		DB:         db,
		UserModel:  model.NewUserModel(db),
		MatchModel: model.NewMatchModel(db),
	}
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "甲"}, {Id: 2, Nickname: "乙"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	result := 1
	opponentID := int64(2)
	completedAt := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	if err := db.Create(&model.Match{Id: 10, UserId: 1, OpponentId: &opponentID, OpponentName: "乙", GameType: 3, Status: 2, Result: &result, MyScore: 7, OpponentScore: 5, MatchTime: completedAt.Add(-time.Hour), CompletedAt: &completedAt}).Error; err != nil {
		t.Fatalf("seed completed match: %v", err)
	}
	if err := db.Create(&model.Match{Id: 11, UserId: 1, OpponentId: &opponentID, OpponentName: "乙", GameType: 3, Status: 2, Result: &result, MatchTime: completedAt}).Error; err != nil {
		t.Fatalf("seed unrelated match: %v", err)
	}
	winner := 1
	if err := db.Create(&model.MatchRound{MatchId: 10, RoundNo: 1, Winner: &winner}).Error; err != nil {
		t.Fatalf("seed round: %v", err)
	}
	if err := db.Create(&model.MatchAction{MatchId: 10, RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 3}).Error; err != nil {
		t.Fatalf("seed action: %v", err)
	}
	if err := db.Create(&model.MatchAchievement{MatchId: 10, Actor: 1, AchievementType: "break_50", Count: 1}).Error; err != nil {
		t.Fatalf("seed achievement: %v", err)
	}

	match, err := svcCtx.MatchModel.FindById(10)
	if err != nil || match == nil {
		t.Fatalf("find target match: %v", err)
	}
	core, err := BuildCompletedMatchCoreSummary(context.Background(), svcCtx, match)
	if err != nil {
		t.Fatalf("build core: %v", err)
	}
	if core.Match.Id != 10 || core.Player1 == nil || core.Player1.Nickname != "甲" || core.Player2 == nil || core.Player2.Nickname != "乙" {
		t.Fatalf("unexpected core participants: %+v", core)
	}
	if len(core.Rounds) != 1 || core.Rounds[0].MatchId != 10 || len(core.Actions) != 1 || core.Actions[0].MatchId != 10 {
		t.Fatalf("core must only include target match actions/rounds: %+v", core)
	}
	if core.Achievements.Break50 != 1 {
		t.Fatalf("core must include target achievements: %+v", core.Achievements)
	}
	viewer := core.ViewerPerspective(2)
	if viewer.IsPlayer1 || viewer.MyName != "乙" || viewer.OpponentName != "甲" || viewer.MyScore != 5 || viewer.OpponentScore != 7 {
		t.Fatalf("unexpected player-two perspective: %+v", viewer)
	}
}
