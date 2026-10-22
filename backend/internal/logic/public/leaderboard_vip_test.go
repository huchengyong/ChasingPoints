package public

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

// 默认成长门槛 10/40/120/300：45 点为 Lv3，300 点为 Lv5。
const (
	vipTestLv3Points = 45
	vipTestLv5Points = 300
)

func openVipLeaderboardDB(t *testing.T, models ...any) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("prepare vip leaderboard schema: %v", err)
	}
	if err := db.Create(&model.RankConfig{Level: 1, Name: "新手"}).Error; err != nil {
		t.Fatalf("seed rank config: %v", err)
	}
	return db
}

// seedVipLeaderboard 写入 5 名用户：1 号 Lv3 会员，2 号无档案会员（Lv1），3 号已到期，4 号从未开通，5 号 Lv5 会员但没有排位记录。
// 1–4 号按分数依次排名，前三名是 1–3 号，第 4 名在列表里。
func seedVipLeaderboard(t *testing.T, db *gorm.DB) {
	t.Helper()
	future := time.Now().Add(24 * time.Hour)
	past := time.Now().Add(-time.Hour)
	if err := db.Create(&[]model.User{
		{Id: 1, Nickname: "甲", MemberExpiresAt: &future},
		{Id: 2, Nickname: "乙", MemberExpiresAt: &future},
		{Id: 3, Nickname: "丙", MemberExpiresAt: &past},
		{Id: 4, Nickname: "丁"},
		{Id: 5, Nickname: "戊", MemberExpiresAt: &future},
	}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Create(&[]model.MemberGrowthProfile{
		{UserId: 1, GrowthPoints: vipTestLv3Points, GrowthLevel: 1},
		{UserId: 3, GrowthPoints: vipTestLv5Points, GrowthLevel: 5},
		{UserId: 5, GrowthPoints: vipTestLv5Points, GrowthLevel: 5},
	}).Error; err != nil {
		t.Fatalf("seed growth profiles: %v", err)
	}
	if err := db.Create(&[]model.UserRanking{
		{UserId: 1, GameType: 3, RankLevel: 1, RankScore: 1000, TotalWins: 1},
		{UserId: 2, GameType: 3, RankLevel: 1, RankScore: 900, TotalWins: 1},
		{UserId: 3, GameType: 3, RankLevel: 1, RankScore: 800, TotalWins: 1},
		{UserId: 4, GameType: 3, RankLevel: 1, RankScore: 700, TotalWins: 1},
	}).Error; err != nil {
		t.Fatalf("seed rankings: %v", err)
	}
}

func vipRequestContext(userID int64) (context.Context, *observability.RequestMetrics) {
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.WithValue(context.Background(), "user_id", userID), metrics)
	return ctx, metrics
}

func getVipLeaderboard(t *testing.T, db *gorm.DB, redisClient *redis.Client, viewerID int64) *types.GetLeaderboardResp {
	t.Helper()
	ctx, _ := vipRequestContext(viewerID)
	resp, err := NewGetLeaderboardLogic(ctx, &svc.ServiceContext{DB: db, Redis: redisClient, RankingModel: model.NewRankingModel(db), UserModel: model.NewUserModel(db)}).
		GetLeaderboard(&types.GetLeaderboardReq{GameType: 3, Page: 1, PageSize: 20})
	if err != nil || !resp.Success {
		t.Fatalf("get leaderboard: resp=%#v err=%v", resp, err)
	}
	return resp
}

func assertVipLevels(t *testing.T, label string, items []types.LeaderboardItem, want map[int64]int) {
	t.Helper()
	for _, item := range items {
		expected, ok := want[item.UserId]
		if !ok {
			t.Fatalf("%s: unexpected user %d", label, item.UserId)
		}
		if item.VipLevel != expected {
			t.Fatalf("%s: user %d vip_level=%d want %d", label, item.UserId, item.VipLevel, expected)
		}
	}
}

func TestLeaderboardPageReturnsVipLevelsForRowsAndViewer(t *testing.T) {
	db := openVipLeaderboardDB(t, &model.User{}, &model.UserRanking{}, &model.RankConfig{}, &model.MemberGrowthProfile{}, &model.MemberRightsConfig{})
	seedVipLeaderboard(t, db)

	resp := getVipLeaderboard(t, db, nil, 1)
	assertVipLevels(t, "top three", resp.TopThree, map[int64]int{1: 3, 2: 1, 3: 0})
	assertVipLevels(t, "list", resp.List, map[int64]int{4: 0})
	if len(resp.TopThree) != 3 || len(resp.List) != 1 {
		t.Fatalf("unexpected page shape: top=%d list=%d", len(resp.TopThree), len(resp.List))
	}
	if resp.MyRanking == nil || resp.MyRanking.UserId != 1 || resp.MyRanking.VipLevel != 3 {
		t.Fatalf("ranked member viewer must carry vip_level: %+v", resp.MyRanking)
	}

	expired := getVipLeaderboard(t, db, nil, 3)
	if expired.MyRanking == nil || expired.MyRanking.UserId != 3 || expired.MyRanking.VipLevel != 0 {
		t.Fatalf("expired viewer must be vip_level 0: %+v", expired.MyRanking)
	}

	unranked := getVipLeaderboard(t, db, nil, 5)
	if unranked.MyRanking == nil || unranked.MyRanking.UserId != 5 || unranked.MyRanking.Rank != 0 || unranked.MyRanking.VipLevel != 5 {
		t.Fatalf("unranked member viewer must carry vip_level: %+v", unranked.MyRanking)
	}
}

func TestLeaderboardSummaryReturnsVipLevelsForTopThreeAndViewer(t *testing.T) {
	db := openVipLeaderboardDB(t, &model.User{}, &model.UserRanking{}, &model.RankConfig{}, &model.MemberGrowthProfile{}, &model.MemberRightsConfig{})
	seedVipLeaderboard(t, db)

	ctx, _ := vipRequestContext(2)
	resp, err := NewGetLeaderboardSummaryLogic(ctx, &svc.ServiceContext{DB: db, RankingModel: model.NewRankingModel(db), UserModel: model.NewUserModel(db)}).
		GetLeaderboardSummary(&types.GetLeaderboardSummaryReq{GameType: 3})
	if err != nil || !resp.Success {
		t.Fatalf("get leaderboard summary: resp=%#v err=%v", resp, err)
	}
	assertVipLevels(t, "summary top three", resp.TopThree, map[int64]int{1: 3, 2: 1, 3: 0})
	if resp.MyRanking == nil || resp.MyRanking.UserId != 2 || resp.MyRanking.VipLevel != 1 {
		t.Fatalf("summary viewer must carry vip_level: %+v", resp.MyRanking)
	}
}

func TestLeaderboardStillSucceedsWhenVipLookupFails(t *testing.T) {
	// 不迁移成长档案表：有效会员的成长值查询必然失败。
	db := openVipLeaderboardDB(t, &model.User{}, &model.UserRanking{}, &model.RankConfig{})
	seedVipLeaderboardWithoutProfiles(t, db)

	resp := getVipLeaderboard(t, db, nil, 1)
	if len(resp.TopThree) != 3 || resp.MyRanking == nil || resp.MyRanking.UserId != 1 {
		t.Fatalf("leaderboard must still return rows and viewer rank: %+v", resp)
	}
	assertVipLevels(t, "degraded top three", resp.TopThree, map[int64]int{1: 0, 2: 0, 3: 0})
	if resp.MyRanking.VipLevel != 0 {
		t.Fatalf("failed vip lookup must degrade to 0: %+v", resp.MyRanking)
	}
}

func seedVipLeaderboardWithoutProfiles(t *testing.T, db *gorm.DB) {
	t.Helper()
	future := time.Now().Add(24 * time.Hour)
	if err := db.Create(&[]model.User{
		{Id: 1, Nickname: "甲", MemberExpiresAt: &future},
		{Id: 2, Nickname: "乙", MemberExpiresAt: &future},
		{Id: 3, Nickname: "丙"},
	}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Create(&[]model.UserRanking{
		{UserId: 1, GameType: 3, RankLevel: 1, RankScore: 1000, TotalWins: 1},
		{UserId: 2, GameType: 3, RankLevel: 1, RankScore: 900, TotalWins: 1},
		{UserId: 3, GameType: 3, RankLevel: 1, RankScore: 800, TotalWins: 1},
	}).Error; err != nil {
		t.Fatalf("seed rankings: %v", err)
	}
}

func TestLeaderboardVipLevelFollowsSharedCacheCycle(t *testing.T) {
	db := openVipLeaderboardDB(t, &model.User{}, &model.UserRanking{}, &model.RankConfig{}, &model.MemberGrowthProfile{}, &model.MemberRightsConfig{})
	seedVipLeaderboard(t, db)
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	ResetLeaderboardCacheForTest()

	first := getVipLeaderboard(t, db, client, 4)
	if first.TopThree[1].UserId != 2 || first.TopThree[1].VipLevel != 1 {
		t.Fatalf("member row must show the badge before expiry: %+v", first.TopThree[1])
	}

	if err := db.Model(&model.User{}).Where("id = ?", 2).Update("member_expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatalf("expire member: %v", err)
	}
	cached := getVipLeaderboard(t, db, client, 4)
	if cached.TopThree[1].VipLevel != 1 {
		t.Fatalf("within the cache cycle the shared payload may still carry the badge: %+v", cached.TopThree[1])
	}

	mr.FastForward(leaderboardCacheTTL + time.Second)
	reloaded := getVipLeaderboard(t, db, client, 4)
	if reloaded.TopThree[1].UserId != 2 || reloaded.TopThree[1].VipLevel != 0 {
		t.Fatalf("after one cache cycle the expired member must lose the badge: %+v", reloaded.TopThree[1])
	}
}

func TestLeaderboardViewerVipLevelIsNeverSharedThroughCache(t *testing.T) {
	db := openVipLeaderboardDB(t, &model.User{}, &model.UserRanking{}, &model.RankConfig{}, &model.MemberGrowthProfile{}, &model.MemberRightsConfig{})
	seedVipLeaderboard(t, db)
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	ResetLeaderboardCacheForTest()

	member := getVipLeaderboard(t, db, client, 1)
	other := getVipLeaderboard(t, db, client, 4)
	if member.MyRanking == nil || member.MyRanking.VipLevel != 3 || other.MyRanking == nil || other.MyRanking.UserId != 4 || other.MyRanking.VipLevel != 0 {
		t.Fatalf("my_ranking must be resolved per viewer: member=%+v other=%+v", member.MyRanking, other.MyRanking)
	}
}

func TestLeaderboardSummaryVipQueriesDoNotGrowWithRows(t *testing.T) {
	one := vipLeaderboardSummaryQueryCount(t, 1)
	hundred := vipLeaderboardSummaryQueryCount(t, 100)
	// 5 次榜单查询，加上榜单行与本人排名各 3 次：用户、成长值、成长规则配置。
	if one != 11 || hundred != 11 {
		t.Fatalf("vip lookups must be a fixed number of queries: one=%d hundred=%d", one, hundred)
	}
}

func vipLeaderboardSummaryQueryCount(t *testing.T, rows int) int64 {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+fmt.Sprintf("-%d?mode=memory&cache=shared", rows)), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserRanking{}, &model.RankConfig{}, &model.MemberGrowthProfile{}, &model.MemberRightsConfig{}); err != nil {
		t.Fatalf("prepare vip query schema: %v", err)
	}
	if err := db.Create(&model.RankConfig{Level: 1, Name: "新手"}).Error; err != nil {
		t.Fatalf("seed rank config: %v", err)
	}
	future := time.Now().Add(24 * time.Hour)
	users := make([]model.User, 0, rows)
	profiles := make([]model.MemberGrowthProfile, 0, rows)
	rankings := make([]model.UserRanking, 0, rows)
	for index := 1; index <= rows; index++ {
		users = append(users, model.User{Id: int64(index), Nickname: fmt.Sprintf("用户%d", index), MemberExpiresAt: &future})
		profiles = append(profiles, model.MemberGrowthProfile{UserId: int64(index), GrowthPoints: index})
		rankings = append(rankings, model.UserRanking{UserId: int64(index), GameType: 3, RankLevel: 1, RankScore: 10000 - index, TotalWins: 1})
	}
	if err := db.CreateInBatches(&users, 100).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.CreateInBatches(&profiles, 100).Error; err != nil {
		t.Fatalf("seed profiles: %v", err)
	}
	if err := db.CreateInBatches(&rankings, 100).Error; err != nil {
		t.Fatalf("seed rankings: %v", err)
	}
	ctx, metrics := vipRequestContext(1)
	resp, err := NewGetLeaderboardSummaryLogic(ctx, &svc.ServiceContext{DB: db, RankingModel: model.NewRankingModel(db)}).GetLeaderboardSummary(&types.GetLeaderboardSummaryReq{GameType: 3})
	if err != nil || !resp.Success || resp.MyRanking == nil || resp.MyRanking.VipLevel == 0 || resp.TopThree[0].VipLevel == 0 {
		t.Fatalf("get leaderboard summary: resp=%#v err=%v", resp, err)
	}
	return metrics.Snapshot().SQLCount
}
