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
