package season

import (
	"context"
	"fmt"
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

func TestSeasonLeaderboardCacheSharesPublicPageAndProfileVersionInvalidates(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Season{}, &model.SeasonRecord{}, &model.User{}); err != nil {
		t.Fatalf("prepare season cache schema: %v", err)
	}
	if err := db.Create(&model.Season{Id: 1, Name: "赛季", StartDate: time.Now().AddDate(0, -1, 0), EndDate: time.Now().AddDate(0, 1, 0), Status: 1}).Error; err != nil {
		t.Fatalf("seed season: %v", err)
	}
	if err := db.Create(&model.User{Id: 1, Nickname: "旧昵称"}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&model.SeasonRecord{SeasonId: 1, UserId: 1, GameType: 3, EndRankScore: 1000, MatchesPlayed: 1, Wins: 1}).Error; err != nil {
		t.Fatalf("seed season record: %v", err)
	}
	miniRedis := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	ResetSeasonCacheForTest()
	load := func() (*types.GetSeasonLeaderboardResp, int64) {
		metrics := observability.NewRequestMetrics(time.Now())
		ctx := observability.WithRequestMetrics(context.Background(), metrics)
		requestDB := db.WithContext(ctx)
		resp, err := NewGetSeasonLeaderboardLogic(ctx, &svc.ServiceContext{Redis: redisClient, SeasonModel: model.NewSeasonModel(requestDB), SeasonRecordModel: model.NewSeasonRecordModel(requestDB)}).
			GetSeasonLeaderboard(&types.GetSeasonLeaderboardReq{SeasonId: 1, GameType: 3, Page: 1, PageSize: 20})
		if err != nil || !resp.Success {
			t.Fatalf("get cached season leaderboard: resp=%#v err=%v", resp, err)
		}
		return resp, metrics.Snapshot().SQLCount
	}
	first, firstQueries := load()
	second, secondQueries := load()
	if firstQueries != 3 || secondQueries != 0 || first.List[0].Nickname != "旧昵称" || second.List[0].Nickname != "旧昵称" {
		t.Fatalf("unexpected shared season cache: first=%+v second=%+v queries=%d/%d", first, second, firstQueries, secondQueries)
	}
	if err := db.Model(&model.User{}).Where("id = ?", 1).Update("nickname", "新昵称").Error; err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if err := BumpSeasonLeaderboardProfileVersion(context.Background(), &svc.ServiceContext{Redis: redisClient}); err != nil {
		t.Fatalf("bump season profile version: %v", err)
	}
	third, thirdQueries := load()
	if thirdQueries != 3 || third.List[0].Nickname != "新昵称" {
		t.Fatalf("profile version must reload season page: resp=%+v queries=%d", third, thirdQueries)
	}
}

func TestSeasonLeaderboardUsesJoinedProfilesWithConstantQueries(t *testing.T) {
	oneQueries := seasonLeaderboardQueryCount(t, 1)
	hundredQueries := seasonLeaderboardQueryCount(t, 100)
	if oneQueries != 3 || hundredQueries != 3 {
		t.Fatalf("season leaderboard must use season lookup, COUNT and joined page query: one=%d hundred=%d", oneQueries, hundredQueries)
	}
}

func seasonLeaderboardQueryCount(t *testing.T, recordCount int) int {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+fmt.Sprintf("-%d?mode=memory&cache=shared", recordCount)), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Season{}, &model.SeasonRecord{}, &model.User{}); err != nil {
		t.Fatalf("prepare season leaderboard schema: %v", err)
	}
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&model.Season{Id: 1, Name: "八月赛季", StartDate: start, EndDate: start.AddDate(0, 1, -1), Status: 1}).Error; err != nil {
		t.Fatalf("seed season: %v", err)
	}
	users := make([]model.User, 0, recordCount)
	records := make([]model.SeasonRecord, 0, recordCount)
	for i := 1; i <= recordCount; i++ {
		userID := int64(i)
		users = append(users, model.User{Id: userID, Nickname: fmt.Sprintf("选手%d", i), Avatar: fmt.Sprintf("%d.png", i)})
		records = append(records, model.SeasonRecord{Id: userID, SeasonId: 1, UserId: userID, GameType: 3, EndRankScore: i, PeakRankScore: i + 1, MatchesPlayed: 3, Wins: 2})
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("seed records: %v", err)
	}
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.Background(), metrics)
	svcCtx := &svc.ServiceContext{
		SeasonModel:       model.NewSeasonModel(db.WithContext(ctx)),
		SeasonRecordModel: model.NewSeasonRecordModel(db.WithContext(ctx)),
	}
	resp, err := NewGetSeasonLeaderboardLogic(ctx, svcCtx).GetSeasonLeaderboard(&types.GetSeasonLeaderboardReq{SeasonId: 1, GameType: 3, Page: 1, PageSize: 100})
	if err != nil || !resp.Success || resp.Total != int64(recordCount) || len(resp.List) != recordCount {
		t.Fatalf("get leaderboard: resp=%#v err=%v", resp, err)
	}
	if resp.List[0].UserId != int64(recordCount) || resp.List[0].Nickname != fmt.Sprintf("选手%d", recordCount) {
		t.Fatalf("joined player profile missing: %#v", resp.List[0])
	}
	return int(metrics.Snapshot().SQLCount)
}
