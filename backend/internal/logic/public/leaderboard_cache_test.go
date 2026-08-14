package public

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLocalLeaderboardLatencyProbe(t *testing.T) {
	if os.Getenv("RUN_LOCAL_PERFORMANCE_PROBE") != "1" {
		t.Skip("set RUN_LOCAL_PERFORMANCE_PROBE=1 to record local cold/hot leaderboard latency")
	}
	db, redisClient := newLeaderboardCacheTestDependencies(t)
	users := make([]model.User, 0, 996)
	rankings := make([]model.UserRanking, 0, 996)
	for userID := 5; userID <= 1000; userID++ {
		users = append(users, model.User{Id: int64(userID), Nickname: fmt.Sprintf("用户%d", userID)})
		rankings = append(rankings, model.UserRanking{UserId: int64(userID), GameType: 3, RankLevel: 1, RankScore: 1000 - userID, TotalWins: 1})
	}
	if err := db.CreateInBatches(&users, 200).Error; err != nil {
		t.Fatalf("seed probe users: %v", err)
	}
	if err := db.CreateInBatches(&rankings, 200).Error; err != nil {
		t.Fatalf("seed probe rankings: %v", err)
	}
	ResetLeaderboardCacheForTest()
	cold := make([]time.Duration, 0, 100)
	var sample *types.GetLeaderboardResp
	for index := 0; index < 100; index++ {
		if err := BumpLeaderboardCacheVersion(context.Background(), &svc.ServiceContext{Redis: redisClient}, 3); err != nil {
			t.Fatalf("bump probe version: %v", err)
		}
		started := time.Now()
		sample, _ = loadCachedLeaderboardForViewer(t, db, redisClient, 1)
		cold = append(cold, time.Since(started))
	}
	hot := make([]time.Duration, 0, 500)
	for index := 0; index < 500; index++ {
		started := time.Now()
		sample, _ = loadCachedLeaderboardForViewer(t, db, redisClient, 1)
		hot = append(hot, time.Since(started))
	}
	payload, _ := json.Marshal(sample)
	metrics := LeaderboardCacheMetrics()
	t.Logf("local leaderboard probe rows=1000 response_bytes=%d cold_p50=%s cold_p95=%s cold_p99=%s hot_p50=%s hot_p95=%s hot_p99=%s hits=%d misses=%d fallbacks=%d",
		len(payload), latencyPercentile(cold, 50), latencyPercentile(cold, 95), latencyPercentile(cold, 99),
		latencyPercentile(hot, 50), latencyPercentile(hot, 95), latencyPercentile(hot, 99), metrics.Hits, metrics.Misses, metrics.Fallbacks)
}

func latencyPercentile(values []time.Duration, percentile int) time.Duration {
	copyValues := append([]time.Duration(nil), values...)
	sort.Slice(copyValues, func(i, j int) bool { return copyValues[i] < copyValues[j] })
	index := (len(copyValues)*percentile + 99) / 100
	if index <= 0 {
		index = 1
	}
	return copyValues[index-1]
}

func TestLeaderboardCacheRepairsCorruptJSONAndCollapsesColdMisses(t *testing.T) {
	miniRedis := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	ResetLeaderboardCacheForTest()
	key := "leaderboard:page:v0:game:3:page:1:size:20"
	if err := client.Set(context.Background(), key, "{bad-json", time.Minute).Err(); err != nil {
		t.Fatalf("seed corrupt leaderboard cache: %v", err)
	}
	var loads atomic.Int64
	loader := func() (*leaderboardSharedPayload, error) {
		loads.Add(1)
		time.Sleep(20 * time.Millisecond)
		return &leaderboardSharedPayload{Total: 1, TopThree: []types.LeaderboardItem{{UserId: 1}}}, nil
	}
	const callers = 20
	start := make(chan struct{})
	var wait sync.WaitGroup
	for index := 0; index < callers; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			value, err := loadLeaderboardShared(context.Background(), &svc.ServiceContext{Redis: client}, "page", 3, 1, 20, loader)
			if err != nil || value.Total != 1 {
				t.Errorf("load shared leaderboard: value=%+v err=%v", value, err)
			}
		}()
	}
	close(start)
	wait.Wait()
	if loads.Load() != 1 {
		t.Fatalf("hot miss must execute one loader, got %d", loads.Load())
	}
	metrics := LeaderboardCacheMetrics()
	if metrics.DecodeErrors == 0 || metrics.Fallbacks != 1 {
		t.Fatalf("corrupt leaderboard cache must fall back and repair: %+v", metrics)
	}
}

func TestLeaderboardCacheFallsBackWhenRedisIsUnavailable(t *testing.T) {
	miniRedis := miniredis.RunT(t)
	address := miniRedis.Addr()
	miniRedis.Close()
	client := redis.NewClient(&redis.Options{Addr: address, MaxRetries: 0, DialTimeout: 20 * time.Millisecond, ReadTimeout: 20 * time.Millisecond, WriteTimeout: 20 * time.Millisecond})
	t.Cleanup(func() { _ = client.Close() })
	ResetLeaderboardCacheForTest()
	value, err := loadLeaderboardShared(context.Background(), &svc.ServiceContext{Redis: client}, "summary", 3, 0, 3, func() (*leaderboardSharedPayload, error) {
		return &leaderboardSharedPayload{Total: 1}, nil
	})
	if err != nil || value.Total != 1 {
		t.Fatalf("Redis outage must use authoritative leaderboard loader: value=%+v err=%v", value, err)
	}
	metrics := LeaderboardCacheMetrics()
	if metrics.RedisErrors == 0 || metrics.Fallbacks != 1 || metrics.WriteErrors == 0 {
		t.Fatalf("leaderboard Redis outage must be observable: %+v", metrics)
	}
}

func TestLeaderboardRedisCacheSharesOnlyPublicPageData(t *testing.T) {
	db, redisClient := newLeaderboardCacheTestDependencies(t)
	ResetLeaderboardCacheForTest()

	first, firstQueries := loadCachedLeaderboardForViewer(t, db, redisClient, 1)
	second, secondQueries := loadCachedLeaderboardForViewer(t, db, redisClient, 2)
	if first.MyRanking == nil || first.MyRanking.UserId != 1 || second.MyRanking == nil || second.MyRanking.UserId != 2 {
		t.Fatalf("viewer rankings must remain independent: first=%+v second=%+v", first.MyRanking, second.MyRanking)
	}
	if firstQueries != 7 || secondQueries != 3 {
		t.Fatalf("shared page must be cached while personal rank stays live: first=%d second=%d", firstQueries, secondQueries)
	}

	if err := db.Model(&model.UserRanking{}).Where("user_id = ? AND game_type = ?", 2, 3).Update("rank_score", 1200).Error; err != nil {
		t.Fatalf("update ranking: %v", err)
	}
	if err := BumpLeaderboardCacheVersion(context.Background(), &svc.ServiceContext{Redis: redisClient}, 3); err != nil {
		t.Fatalf("bump leaderboard version: %v", err)
	}
	third, thirdQueries := loadCachedLeaderboardForViewer(t, db, redisClient, 2)
	if thirdQueries != 7 || len(third.TopThree) == 0 || third.TopThree[0].UserId != 2 || third.MyRanking == nil || third.MyRanking.Rank != 1 {
		t.Fatalf("version bump must reload shared page and combine current viewer rank: resp=%+v queries=%d", third, thirdQueries)
	}
	metrics := LeaderboardCacheMetrics()
	if metrics.Hits != 1 || metrics.Misses != 2 || metrics.Fallbacks != 2 {
		t.Fatalf("unexpected leaderboard cache metrics: %+v", metrics)
	}
}

func TestLeaderboardSummaryCacheAlsoKeepsViewerRankSeparate(t *testing.T) {
	db, redisClient := newLeaderboardCacheTestDependencies(t)
	ResetLeaderboardCacheForTest()

	first, firstQueries := loadCachedLeaderboardSummaryForViewer(t, db, redisClient, 1)
	second, secondQueries := loadCachedLeaderboardSummaryForViewer(t, db, redisClient, 2)
	if first.MyRanking == nil || first.MyRanking.UserId != 1 || second.MyRanking == nil || second.MyRanking.UserId != 2 {
		t.Fatalf("summary viewer rankings must remain independent: first=%+v second=%+v", first.MyRanking, second.MyRanking)
	}
	if firstQueries != 5 || secondQueries != 3 {
		t.Fatalf("summary cache must save only config/top-three queries: first=%d second=%d", firstQueries, secondQueries)
	}
}

func newLeaderboardCacheTestDependencies(t *testing.T) (*gorm.DB, *redis.Client) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserRanking{}, &model.RankConfig{}); err != nil {
		t.Fatalf("prepare leaderboard cache schema: %v", err)
	}
	if err := db.Create(&model.RankConfig{Level: 1, Name: "新手"}).Error; err != nil {
		t.Fatalf("seed rank config: %v", err)
	}
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "甲"}, {Id: 2, Nickname: "乙"}, {Id: 3, Nickname: "丙"}, {Id: 4, Nickname: "丁"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Create(&[]model.UserRanking{
		{UserId: 1, GameType: 3, RankLevel: 1, RankScore: 1000, TotalWins: 1},
		{UserId: 2, GameType: 3, RankLevel: 1, RankScore: 900, TotalWins: 1},
		{UserId: 3, GameType: 3, RankLevel: 1, RankScore: 800, TotalWins: 1},
		{UserId: 4, GameType: 3, RankLevel: 1, RankScore: 700, TotalWins: 1},
	}).Error; err != nil {
		t.Fatalf("seed rankings: %v", err)
	}
	miniRedis := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return db, client
}

func loadCachedLeaderboardForViewer(t *testing.T, db *gorm.DB, redisClient *redis.Client, userID int64) (*types.GetLeaderboardResp, int64) {
	t.Helper()
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.WithValue(context.Background(), "user_id", userID), metrics)
	resp, err := NewGetLeaderboardLogic(ctx, &svc.ServiceContext{DB: db, Redis: redisClient, RankingModel: model.NewRankingModel(db), UserModel: model.NewUserModel(db)}).GetLeaderboard(&types.GetLeaderboardReq{GameType: 3, Page: 1, PageSize: 20})
	if err != nil || !resp.Success {
		t.Fatalf("get leaderboard: resp=%#v err=%v", resp, err)
	}
	return resp, metrics.Snapshot().SQLCount
}

func loadCachedLeaderboardSummaryForViewer(t *testing.T, db *gorm.DB, redisClient *redis.Client, userID int64) (*types.GetLeaderboardSummaryResp, int64) {
	t.Helper()
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.WithValue(context.Background(), "user_id", userID), metrics)
	resp, err := NewGetLeaderboardSummaryLogic(ctx, &svc.ServiceContext{DB: db, Redis: redisClient, RankingModel: model.NewRankingModel(db)}).GetLeaderboardSummary(&types.GetLeaderboardSummaryReq{GameType: 3})
	if err != nil || !resp.Success {
		t.Fatalf("get leaderboard summary: resp=%#v err=%v", resp, err)
	}
	return resp, metrics.Snapshot().SQLCount
}
