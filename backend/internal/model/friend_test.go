package model

import (
	"testing"

	"chasing_points/internal/testsupport"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newFriendModelTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := testsupport.PrepareFriendSchema(db); err != nil {
		t.Fatalf("prepare friend schema: %v", err)
	}

	return db
}

func seedFriendTestUser(t *testing.T, userModel *UserModel, id int64, nickname string) {
	t.Helper()

	if err := userModel.Create(&User{
		Id:       id,
		Nickname: nickname,
		Status:   1,
	}); err != nil {
		t.Fatalf("create user %d: %v", id, err)
	}
}

func TestFriendModelBlacklistFriendRemovesFriendshipAndPendingRequests(t *testing.T) {
	db := newFriendModelTestDB(t)
	friendModel := NewFriendModel(db)
	userModel := NewUserModel(db)

	seedFriendTestUser(t, userModel, 101, "球友甲")
	seedFriendTestUser(t, userModel, 202, "球友乙")

	if err := friendModel.AddFriend(101, 202); err != nil {
		t.Fatalf("add friend: %v", err)
	}
	if _, err := friendModel.SendRequest(101, 202, "一起打球"); err != nil {
		t.Fatalf("send pending request: %v", err)
	}
	if _, err := friendModel.SendRequest(202, 101, "回个好友"); err != nil {
		t.Fatalf("send reverse pending request: %v", err)
	}

	if err := friendModel.BlacklistFriend(101, 202); err != nil {
		t.Fatalf("blacklist friend: %v", err)
	}

	areFriends, err := friendModel.AreFriends(101, 202)
	if err != nil {
		t.Fatalf("check friendship: %v", err)
	}
	if areFriends {
		t.Fatal("expected friendship removed after blacklisting")
	}

	hasBlacklistRelation, err := friendModel.HasBlacklistRelation(101, 202)
	if err != nil {
		t.Fatalf("check blacklist relation: %v", err)
	}
	if !hasBlacklistRelation {
		t.Fatal("expected blacklist relation to exist")
	}

	var pendingCount int64
	if err := db.Model(&FriendRequest{}).
		Where("status = 0").
		Where("(from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?)", 101, 202, 202, 101).
		Count(&pendingCount).Error; err != nil {
		t.Fatalf("count pending requests: %v", err)
	}
	if pendingCount != 0 {
		t.Fatalf("expected pending requests cleared, got %d", pendingCount)
	}
}

func TestFriendModelSearchUsersHidesBlacklistedPairs(t *testing.T) {
	db := newFriendModelTestDB(t)
	friendModel := NewFriendModel(db)
	userModel := NewUserModel(db)

	seedFriendTestUser(t, userModel, 101, "搜索用户")
	seedFriendTestUser(t, userModel, 202, "球友可见")
	seedFriendTestUser(t, userModel, 303, "球友被我拉黑")
	seedFriendTestUser(t, userModel, 404, "球友拉黑我")

	if err := friendModel.BlacklistFriend(101, 303); err != nil {
		t.Fatalf("blacklist outgoing relation: %v", err)
	}
	if err := friendModel.BlacklistFriend(404, 101); err != nil {
		t.Fatalf("blacklist incoming relation: %v", err)
	}

	list, err := friendModel.SearchUsers(101, "球友", 20)
	if err != nil {
		t.Fatalf("search users: %v", err)
	}

	if len(list) != 1 {
		t.Fatalf("expected 1 visible user, got %#v", list)
	}
	if list[0].Id != 202 {
		t.Fatalf("expected visible user 202, got %#v", list[0])
	}
}

func TestFriendModelSearchEscapesLikeWildcards(t *testing.T) {
	db := newFriendModelTestDB(t)
	friendModel := NewFriendModel(db)
	userModel := NewUserModel(db)

	if err := db.AutoMigrate(&UserRanking{}, &RankConfig{}); err != nil {
		t.Fatalf("prepare ranking schema: %v", err)
	}
	seedFriendTestUser(t, userModel, 101, "搜索用户")
	seedFriendTestUser(t, userModel, 202, "普通球友")
	seedFriendTestUser(t, userModel, 303, "百分百%球友")
	seedFriendTestUser(t, userModel, 404, "下划线_球友")

	percentList, err := friendModel.SearchUsers(101, "%", 20)
	if err != nil {
		t.Fatalf("search percent: %v", err)
	}
	if len(percentList) != 1 || percentList[0].Id != 303 {
		t.Fatalf("percent wildcard must match literal only: %#v", percentList)
	}
	underscoreList, err := friendModel.SearchUsers(101, "_", 20)
	if err != nil {
		t.Fatalf("search underscore: %v", err)
	}
	if len(underscoreList) != 1 || underscoreList[0].Id != 404 {
		t.Fatalf("underscore wildcard must match literal only: %#v", underscoreList)
	}

	profiles, err := friendModel.SearchUserProfiles(101, "%", 20)
	if err != nil {
		t.Fatalf("search profiles percent: %v", err)
	}
	if len(profiles) != 1 || profiles[0].UserId != 303 {
		t.Fatalf("profile search must escape wildcards too: %#v", profiles)
	}
}

func TestFriendModelSearchMatchesPhoneOnlyForElevenDigits(t *testing.T) {
	db := newFriendModelTestDB(t)
	friendModel := NewFriendModel(db)
	userModel := NewUserModel(db)

	seedFriendTestUser(t, userModel, 101, "搜索用户")
	phone := "13800138000"
	if err := userModel.Create(&User{Id: 202, Nickname: "手机号球友", Status: 1, Phone: &phone}); err != nil {
		t.Fatalf("create phone user: %v", err)
	}

	byPhone, err := friendModel.SearchUsers(101, phone, 20)
	if err != nil {
		t.Fatalf("search by phone: %v", err)
	}
	if len(byPhone) != 1 || byPhone[0].Id != 202 {
		t.Fatalf("eleven-digit keyword must match phone exactly: %#v", byPhone)
	}
	partial, err := friendModel.SearchUsers(101, "1380013800", 20)
	if err != nil {
		t.Fatalf("search by partial phone: %v", err)
	}
	if len(partial) != 0 {
		t.Fatalf("partial phone must not match: %#v", partial)
	}
}

func TestFriendModelSearchReportsPendingRequest(t *testing.T) {
	db := newFriendModelTestDB(t)
	friendModel := NewFriendModel(db)
	userModel := NewUserModel(db)

	if err := db.AutoMigrate(&UserRanking{}, &RankConfig{}); err != nil {
		t.Fatalf("prepare ranking schema: %v", err)
	}
	seedFriendTestUser(t, userModel, 101, "搜索用户")
	seedFriendTestUser(t, userModel, 202, "待处理球友")
	if _, err := friendModel.SendRequest(101, 202, "一起打球"); err != nil {
		t.Fatalf("send request: %v", err)
	}

	profiles, err := friendModel.SearchUserProfiles(101, "待处理", 20)
	if err != nil {
		t.Fatalf("search profiles: %v", err)
	}
	if len(profiles) != 1 || !profiles[0].HasPendingRequest {
		t.Fatalf("expected has_pending_request=true: %#v", profiles)
	}
}

func TestFriendAndFollowModelClampPageSize(t *testing.T) {
	db := newFriendModelTestDB(t)
	friendModel := NewFriendModel(db)
	followModel := NewFollowModel(db)
	userModel := NewUserModel(db)
	seedFriendTestUser(t, userModel, 101, "我")

	if err := db.AutoMigrate(&Follow{}); err != nil {
		t.Fatalf("prepare follow schema: %v", err)
	}
	for i := 0; i < 25; i++ {
		if err := db.Create(&Friend{UserId: 101, FriendId: int64(1000 + i), Status: 1}).Error; err != nil {
			t.Fatalf("seed friend: %v", err)
		}
		if _, err := followModel.Follow(101, int64(2000+i)); err != nil {
			t.Fatalf("seed follow: %v", err)
		}
	}

	for _, pageSize := range []int{0, -1, 101, 1000} {
		friends, _, err := friendModel.GetFriendList(101, 1, pageSize)
		if err != nil {
			t.Fatalf("friend list pageSize %d: %v", pageSize, err)
		}
		if len(friends) != 20 {
			t.Fatalf("friend list pageSize %d must fall back to 20, got %d", pageSize, len(friends))
		}
		following, _, err := followModel.GetFollowingList(101, 1, pageSize)
		if err != nil {
			t.Fatalf("following list pageSize %d: %v", pageSize, err)
		}
		if len(following) != 20 {
			t.Fatalf("following list pageSize %d must fall back to 20, got %d", pageSize, len(following))
		}
	}
}
