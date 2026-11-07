package friend

import (
	"context"
	"fmt"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openFriendVipTestDB(t *testing.T, name string, withGrowthProfiles bool) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	models := []any{&model.User{}, &model.Friend{}, &model.UserRanking{}, &model.RankConfig{}, &model.MemberRightsConfig{}}
	if withGrowthProfiles {
		models = append(models, &model.MemberGrowthProfile{})
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("prepare friend vip schema: %v", err)
	}
	if err := db.Create(&model.RankConfig{Level: 1, Name: "新手"}).Error; err != nil {
		t.Fatalf("seed rank config: %v", err)
	}
	return db
}

func getFriendListWithMetrics(t *testing.T, db *gorm.DB, pageSize int) (*types.GetFriendListResp, int64) {
	t.Helper()
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.WithValue(context.Background(), "user_id", int64(1)), metrics)
	resp, err := NewGetFriendListLogic(ctx, &svc.ServiceContext{DB: db}).GetFriendList(&types.GetFriendListReq{Page: 1, PageSize: pageSize})
	if err != nil || !resp.Success {
		t.Fatalf("get friends: resp=%#v err=%v", resp, err)
	}
	return resp, metrics.Snapshot().SQLCount
}

func TestGetFriendListReturnsVipLevelForMembersOnly(t *testing.T) {
	db := openFriendVipTestDB(t, t.Name(), true)
	future := time.Now().Add(24 * time.Hour)
	past := time.Now().Add(-time.Hour)
	base := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	if err := db.Create(&[]model.User{
		{Id: 1, Nickname: "我"},
		{Id: 2, Nickname: "Lv3会员", MemberExpiresAt: &future},
		{Id: 3, Nickname: "无档案会员", MemberExpiresAt: &future},
		{Id: 4, Nickname: "已到期", MemberExpiresAt: &past},
		{Id: 5, Nickname: "免费用户"},
	}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	// 默认成长门槛 10/40/120/300：45 点为 Lv3。已到期用户的成长值保留但不展示。
	if err := db.Create(&[]model.MemberGrowthProfile{
		{UserId: 2, GrowthPoints: 45, GrowthLevel: 1},
		{UserId: 4, GrowthPoints: 300, GrowthLevel: 5},
	}).Error; err != nil {
		t.Fatalf("seed growth profiles: %v", err)
	}
	friends := make([]model.Friend, 0, 4)
	for i, friendID := range []int64{2, 3, 4, 5} {
		friends = append(friends, model.Friend{Id: friendID, UserId: 1, FriendId: friendID, Status: 1, CreatedAt: base.Add(time.Duration(i) * time.Second)})
	}
	if err := db.Create(&friends).Error; err != nil {
		t.Fatalf("seed friends: %v", err)
	}

	resp, _ := getFriendListWithMetrics(t, db, 20)
	want := map[int64]int{2: 3, 3: 1, 4: 0, 5: 0}
	if len(resp.List) != len(want) {
		t.Fatalf("unexpected friend list: %+v", resp.List)
	}
	for _, item := range resp.List {
		if item.VipLevel != want[item.UserId] {
			t.Fatalf("friend %d vip_level=%d want %d", item.UserId, item.VipLevel, want[item.UserId])
		}
	}
}

func TestGetFriendListStillSucceedsWhenVipLookupFails(t *testing.T) {
	// 不迁移成长档案表：有效会员的成长值查询必然失败。
	db := openFriendVipTestDB(t, t.Name(), false)
	future := time.Now().Add(24 * time.Hour)
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "我"}, {Id: 2, Nickname: "会员", MemberExpiresAt: &future}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Create(&model.Friend{Id: 2, UserId: 1, FriendId: 2, Status: 1}).Error; err != nil {
		t.Fatalf("seed friend: %v", err)
	}

	resp, _ := getFriendListWithMetrics(t, db, 20)
	if resp.Total != 1 || len(resp.List) != 1 || resp.List[0].UserId != 2 || resp.List[0].Nickname != "会员" {
		t.Fatalf("friend list must still be returned: %+v", resp)
	}
	if resp.List[0].VipLevel != 0 {
		t.Fatalf("failed vip lookup must degrade to 0: %+v", resp.List[0])
	}
}

func TestGetFriendListVipQueriesDoNotGrowWithFriends(t *testing.T) {
	counts := map[int]int64{}
	for _, friendCount := range []int{1, 100} {
		db := openFriendVipTestDB(t, fmt.Sprintf("%s-%d", t.Name(), friendCount), true)
		future := time.Now().Add(24 * time.Hour)
		users := []model.User{{Id: 1, Nickname: "我"}}
		friends := make([]model.Friend, 0, friendCount)
		profiles := make([]model.MemberGrowthProfile, 0, friendCount)
		for i := range friendCount {
			friendID := int64(i + 2)
			users = append(users, model.User{Id: friendID, Nickname: fmt.Sprintf("好友%d", friendID), MemberExpiresAt: &future})
			friends = append(friends, model.Friend{Id: friendID, UserId: 1, FriendId: friendID, Status: 1})
			profiles = append(profiles, model.MemberGrowthProfile{UserId: friendID, GrowthPoints: i + 1})
		}
		if err := db.CreateInBatches(&users, 100).Error; err != nil {
			t.Fatalf("seed users: %v", err)
		}
		if err := db.CreateInBatches(&friends, 100).Error; err != nil {
			t.Fatalf("seed friends: %v", err)
		}
		if err := db.CreateInBatches(&profiles, 100).Error; err != nil {
			t.Fatalf("seed profiles: %v", err)
		}
		resp, queries := getFriendListWithMetrics(t, db, 100)
		if len(resp.List) != friendCount || resp.List[0].VipLevel == 0 {
			t.Fatalf("friends must carry vip levels: len=%d first=%+v", len(resp.List), resp.List[0])
		}
		counts[friendCount] = queries
	}
	// COUNT 与联表分页各 1 次，会员档位再加用户、成长值、成长规则配置各 1 次。
	if counts[1] != 5 || counts[100] != 5 {
		t.Fatalf("vip lookups must be a fixed number of queries: one=%d hundred=%d", counts[1], counts[100])
	}
}
