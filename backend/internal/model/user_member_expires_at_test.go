package model

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFindMemberExpiresAtByIdsReturnsOnlyLiveUsersWithExpiry(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("prepare user schema: %v", err)
	}
	expires := time.Date(2026, 12, 1, 8, 0, 0, 0, time.UTC)
	// 1100 个用户跨过 500 条一批的分批边界。
	const total = 1100
	users := make([]User, 0, total)
	ids := make([]int64, 0, total)
	for i := 1; i <= total; i++ {
		user := User{Id: int64(i), Nickname: "用户"}
		if i == 7 || i == 600 || i == 1100 {
			e := expires
			user.MemberExpiresAt = &e
		}
		users = append(users, user)
		ids = append(ids, int64(i))
	}
	if err := db.CreateInBatches(&users, 200).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Delete(&User{}, 600).Error; err != nil {
		t.Fatalf("soft delete user: %v", err)
	}

	got, err := NewUserModel(db).FindMemberExpiresAtByIds(ids)
	if err != nil {
		t.Fatalf("find member expires: %v", err)
	}
	if len(got) != 2 || !got[7].Equal(expires) || !got[1100].Equal(expires) {
		t.Fatalf("only live users with an expiry must be returned: %v", got)
	}

	empty, err := NewUserModel(db).FindMemberExpiresAtByIds(nil)
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty ids: %v err=%v", empty, err)
	}
}
