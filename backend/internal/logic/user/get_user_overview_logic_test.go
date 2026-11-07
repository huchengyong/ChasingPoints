package user

import (
	"context"
	"fmt"
	"testing"
	"time"

	"chasing_points/internal/config"
	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	"chasing_points/internal/requestctx"
	"chasing_points/internal/svc"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserOverviewRejectsMissingAuthentication(t *testing.T) {
	resp, err := NewGetUserOverviewLogic(context.Background(), &svc.ServiceContext{}).GetUserOverview()
	if err != nil || resp.Success || len(resp.Availability) != 0 {
		t.Fatalf("unauthenticated user overview must not return data: resp=%#v err=%v", resp, err)
	}
}

func TestUserOverviewReturnsAvailableBlocksAndPartialOptionalFailure(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserCompetitiveStats{}, &model.UserReputationProfile{}, &model.MemberGrowthProfile{}); err != nil {
		t.Fatalf("prepare overview schema: %v", err)
	}
	if err := db.Create(&model.User{Id: 1, Nickname: "我"}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&model.UserCompetitiveStats{UserId: 1, GameType: 0, TotalMatches: 4, Wins: 3, Losses: 1, MaxWinStreak: 2, Revision: 8}).Error; err != nil {
		t.Fatalf("seed stats: %v", err)
	}
	svcCtx := &svc.ServiceContext{
		DB:                         db,
		UserModel:                  model.NewUserModel(db),
		CompetitiveReadModel:       model.NewCompetitiveReadModel(db),
		UserReputationProfileModel: model.NewUserReputationProfileModel(db),
		MemberGrowthProfileModel:   model.NewMemberGrowthProfileModel(db),
		Config:                     config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}},
	}
	resp, err := NewGetUserOverviewLogic(context.WithValue(context.Background(), "user_id", int64(1)), svcCtx).GetUserOverview()
	if err != nil || !resp.Success || resp.Stats == nil || resp.Reputation == nil || resp.MemberStatus == nil {
		t.Fatalf("get overview: resp=%#v err=%v", resp, err)
	}
	if resp.Stats.TotalMatches != 4 || resp.Stats.WinRate != 75 || resp.CompetitiveRevision != 8 {
		t.Fatalf("unexpected overview stats/revision: %+v", resp)
	}
	legacyStats, legacyErr := NewGetUserStatsLogic(context.WithValue(context.Background(), "user_id", int64(1)), svcCtx).GetUserStats()
	if legacyErr != nil || !legacyStats.Success || legacyStats.TotalMatches != resp.Stats.TotalMatches || legacyStats.Wins != resp.Stats.Wins || legacyStats.WinRate != resp.Stats.WinRate {
		t.Fatalf("overview and legacy stats must share the snapshot result: overview=%+v legacy=%+v err=%v", resp.Stats, legacyStats, legacyErr)
	}
	if !resp.Availability["stats"] || !resp.Availability["reputation"] || !resp.Availability["member"] || resp.Availability["favorite_venue_reward"] {
		t.Fatalf("unexpected overview availability: %+v", resp.Availability)
	}
	if len(resp.PartialErrors) != 1 || resp.PartialErrors[0].Scope != "favorite_venue_reward" {
		t.Fatalf("expected only favorite reward partial error: %+v", resp.PartialErrors)
	}
}

func TestUserOverviewUsesFixedQueryCountForOneAndHundredRows(t *testing.T) {
	one := userOverviewQueryCount(t, 1)
	hundred := userOverviewQueryCount(t, 100)
	if one != 3 || hundred != 3 {
		t.Fatalf("user overview must use three fixed core business queries: one=%d hundred=%d", one, hundred)
	}
}

func TestUserOverviewUsesFourBusinessQueriesWithWarmRuntimeConfigs(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.UserCompetitiveStats{},
		&model.ReputationConfig{},
		&model.UserReputationProfile{},
		&model.MemberRightsConfig{},
		&model.MemberGrowthProfile{},
		&model.FavoriteVenueRewardConfig{},
		&model.FavoriteVenueRewardRecord{},
		&model.Venue{},
	); err != nil {
		t.Fatalf("prepare full overview schema: %v", err)
	}
	user := model.User{Id: 1, Nickname: "我", Status: 1}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&model.UserCompetitiveStats{UserId: 1, GameType: 0, TotalMatches: 1, Wins: 1, Revision: 1}).Error; err != nil {
		t.Fatalf("seed competitive stats: %v", err)
	}
	if err := model.NewReputationConfigModel(db).Upsert(model.DefaultReputationConfig()); err != nil {
		t.Fatalf("seed reputation config: %v", err)
	}
	if err := model.NewMemberRightsConfigModel(db).Upsert(model.DefaultMemberRightsConfig()); err != nil {
		t.Fatalf("seed member rights config: %v", err)
	}
	if err := model.NewFavoriteVenueRewardConfigModel(db).Upsert(model.DefaultFavoriteVenueRewardConfig()); err != nil {
		t.Fatalf("seed favorite venue reward config: %v", err)
	}
	miniRedis := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	baseSvc := &svc.ServiceContext{
		DB:                             db,
		Redis:                          redisClient,
		UserModel:                      model.NewUserModel(db),
		CompetitiveReadModel:           model.NewCompetitiveReadModel(db),
		UserReputationProfileModel:     model.NewUserReputationProfileModel(db),
		ReputationConfigModel:          model.NewReputationConfigModel(db),
		MemberGrowthProfileModel:       model.NewMemberGrowthProfileModel(db),
		MemberRightsConfigModel:        model.NewMemberRightsConfigModel(db),
		FavoriteVenueRewardConfigModel: model.NewFavoriteVenueRewardConfigModel(db),
		FavoriteVenueRewardRecordModel: model.NewFavoriteVenueRewardRecordModel(db),
		VenueModel:                     model.NewVenueModel(db),
		Config:                         config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}},
	}
	if _, err := logicx.NewReputationConfigService(baseSvc).GetConfig(); err != nil {
		t.Fatalf("warm reputation config: %v", err)
	}
	if _, err := logicx.NewMemberRightsConfigService(baseSvc).GetConfig(); err != nil {
		t.Fatalf("warm member rights config: %v", err)
	}
	if _, err := logicx.LoadFavoriteVenueRewardConfig(baseSvc); err != nil {
		t.Fatalf("warm favorite venue reward config: %v", err)
	}

	metrics := observability.NewRequestMetrics(time.Now())
	ctx := context.WithValue(context.Background(), "user_id", int64(1))
	ctx = requestctx.WithActiveUser(ctx, &user)
	ctx = observability.WithRequestMetrics(ctx, metrics)
	requestDB := db.WithContext(ctx)
	requestSvc := &svc.ServiceContext{
		DB:                             requestDB,
		Redis:                          redisClient,
		UserModel:                      model.NewUserModel(requestDB),
		CompetitiveReadModel:           model.NewCompetitiveReadModel(requestDB),
		UserReputationProfileModel:     model.NewUserReputationProfileModel(requestDB),
		ReputationConfigModel:          model.NewReputationConfigModel(requestDB),
		MemberGrowthProfileModel:       model.NewMemberGrowthProfileModel(requestDB),
		MemberRightsConfigModel:        model.NewMemberRightsConfigModel(requestDB),
		FavoriteVenueRewardConfigModel: model.NewFavoriteVenueRewardConfigModel(requestDB),
		FavoriteVenueRewardRecordModel: model.NewFavoriteVenueRewardRecordModel(requestDB),
		VenueModel:                     model.NewVenueModel(requestDB),
		Config:                         baseSvc.Config,
	}
	resp, err := NewGetUserOverviewLogic(ctx, requestSvc).GetUserOverview()
	if err != nil || !resp.Success || !resp.Availability["stats"] || !resp.Availability["reputation"] || !resp.Availability["member"] || !resp.Availability["favorite_venue_reward"] {
		t.Fatalf("get full user overview: resp=%#v err=%v", resp, err)
	}
	snapshot := metrics.Snapshot()
	if snapshot.SQLCount != 4 || snapshot.CacheHits != 3 {
		t.Fatalf("warm user overview must use four business SQL queries and three config cache hits: %+v", snapshot)
	}
}

func userOverviewQueryCount(t *testing.T, rows int) int64 {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+fmt.Sprintf("-%d?mode=memory&cache=shared", rows)), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserCompetitiveStats{}, &model.UserReputationProfile{}, &model.MemberGrowthProfile{}); err != nil {
		t.Fatalf("prepare overview query schema: %v", err)
	}
	user := model.User{Id: 1, Nickname: "我"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	stats := make([]model.UserCompetitiveStats, 0, rows)
	stats = append(stats, model.UserCompetitiveStats{UserId: 1, GameType: 0, TotalMatches: rows, Wins: rows, Revision: int64(rows)})
	for index := 1; index < rows; index++ {
		stats = append(stats, model.UserCompetitiveStats{UserId: int64(1000 + index), GameType: 0})
	}
	if err := db.Create(&stats).Error; err != nil {
		t.Fatalf("seed stats: %v", err)
	}
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := context.WithValue(context.Background(), "user_id", int64(1))
	ctx = requestctx.WithActiveUser(ctx, &user)
	ctx = observability.WithRequestMetrics(ctx, metrics)
	requestDB := db.WithContext(ctx)
	resp, err := NewGetUserOverviewLogic(ctx, &svc.ServiceContext{
		DB:                         requestDB,
		UserModel:                  model.NewUserModel(requestDB),
		CompetitiveReadModel:       model.NewCompetitiveReadModel(requestDB),
		UserReputationProfileModel: model.NewUserReputationProfileModel(requestDB),
		MemberGrowthProfileModel:   model.NewMemberGrowthProfileModel(requestDB),
		Config:                     config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}},
	}).GetUserOverview()
	if err != nil || !resp.Success || !resp.Availability["stats"] || !resp.Availability["reputation"] || !resp.Availability["member"] {
		t.Fatalf("get user overview: resp=%#v err=%v", resp, err)
	}
	return metrics.Snapshot().SQLCount
}
