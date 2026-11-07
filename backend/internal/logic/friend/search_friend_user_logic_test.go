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

func TestSearchFriendUsersUsesOneRelationAndRankQuery(t *testing.T) {
	oneQueries := searchFriendQueryCount(t, 1)
	hundredQueries := searchFriendQueryCount(t, 100)
	if oneQueries != 1 || hundredQueries != 1 {
		t.Fatalf("friend search must use one bounded relation/profile query: one=%d hundred=%d", oneQueries, hundredQueries)
	}
}

func TestSearchFriendUsersFiltersFriendsAndBlacklistAndLoadsPendingRelation(t *testing.T) {
	_, svcCtx := prepareSearchFriendTestSvc(t, 5)
	profiles, err := svcCtx.FriendModel.SearchUserProfiles(1, "测试", 20)
	if err != nil {
		t.Fatalf("load candidate profiles: %v", err)
	}
	var pendingFound bool
	for _, profile := range profiles {
		if profile.UserId == 4 {
			pendingFound = profile.HasPendingRequest
		}
		if profile.UserId == 3 {
			t.Fatalf("blacklisted candidate must not be returned: %#v", profiles)
		}
	}
	if !pendingFound {
		t.Fatalf("pending request relation must be loaded with candidate profile: %#v", profiles)
	}
	resp, err := NewSearchFriendUserLogic(context.WithValue(context.Background(), "user_id", int64(1)), svcCtx).SearchFriendUser(&types.SearchUserReq{Keyword: "测试"})
	if err != nil || !resp.Success {
		t.Fatalf("search candidates: resp=%#v err=%v", resp, err)
	}
	for _, item := range resp.List {
		if item.UserId == 2 || item.UserId == 3 || item.IsFriend {
			t.Fatalf("friend/blacklist relation leaked into visible candidate list: %#v", resp.List)
		}
	}
}

func searchFriendQueryCount(t *testing.T, candidateCount int) int {
	t.Helper()
	db, svcCtx := prepareSearchFriendTestSvc(t, candidateCount)
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.WithValue(context.Background(), "user_id", int64(1)), metrics)
	svcCtx.FriendModel = model.NewFriendModel(db.WithContext(ctx))
	resp, err := NewSearchFriendUserLogic(ctx, svcCtx).SearchFriendUser(&types.SearchUserReq{Keyword: "测试"})
	if err != nil || !resp.Success {
		t.Fatalf("search candidates: resp=%#v err=%v", resp, err)
	}
	return int(metrics.Snapshot().SQLCount)
}

func prepareSearchFriendTestSvc(t *testing.T, candidateCount int) (*gorm.DB, *svc.ServiceContext) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+fmt.Sprintf("-%d?mode=memory&cache=shared", candidateCount)), &gorm.Config{
		Logger: observability.NewGormLogger(time.Hour),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Friend{}, &model.FriendRequest{}, &model.FriendBlacklist{}, &model.UserRanking{}, &model.RankConfig{}); err != nil {
		t.Fatalf("prepare search schema: %v", err)
	}
	if err := db.Create(&[]model.RankConfig{{Level: 1, Name: "新手"}, {Level: 2, Name: "进阶"}}).Error; err != nil {
		t.Fatalf("seed configs: %v", err)
	}
	users := []model.User{{Id: 1, Nickname: "我"}}
	rankings := make([]model.UserRanking, 0, candidateCount)
	for i := 0; i < candidateCount; i++ {
		userID := int64(i + 2)
		users = append(users, model.User{Id: userID, Nickname: fmt.Sprintf("测试用户%d", userID)})
		rankings = append(rankings, model.UserRanking{UserId: userID, GameType: 3, RankLevel: 2})
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Create(&rankings).Error; err != nil {
		t.Fatalf("seed rankings: %v", err)
	}
	if candidateCount >= 1 {
		if err := db.Create(&model.Friend{UserId: 1, FriendId: 2, Status: 1}).Error; err != nil {
			t.Fatalf("seed friend: %v", err)
		}
	}
	if candidateCount >= 2 {
		if err := db.Create(&model.FriendBlacklist{UserId: 1, BlockedUserId: 3}).Error; err != nil {
			t.Fatalf("seed blacklist: %v", err)
		}
	}
	if candidateCount >= 3 {
		if err := db.Create(&model.FriendRequest{FromUserId: 4, ToUserId: 1, Status: 0}).Error; err != nil {
			t.Fatalf("seed pending request: %v", err)
		}
	}
	return db, &svc.ServiceContext{FriendModel: model.NewFriendModel(db)}
}
