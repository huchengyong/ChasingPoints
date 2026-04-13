package admin

import (
	"context"
	"testing"
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAdminUserListTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("prepare user schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:        db,
		UserModel: model.NewUserModel(db),
	}
}

func TestAdminGetUserListIncludesMemberStatusAndExpiry(t *testing.T) {
	svcCtx := newAdminUserListTestSvc(t)
	activeExpiresAt := logicx.NowUTC8().Add(24 * time.Hour)
	expiredAt := logicx.NowUTC8().Add(-24 * time.Hour)
	users := []model.User{
		{Id: 1001, Nickname: "会员用户", Status: 1, MemberExpiresAt: &activeExpiresAt},
		{Id: 1002, Nickname: "过期用户", Status: 1, MemberExpiresAt: &expiredAt},
		{Id: 1003, Nickname: "普通用户", Status: 1},
	}
	for _, user := range users {
		u := user
		if err := svcCtx.UserModel.Create(&u); err != nil {
			t.Fatalf("create user %d: %v", u.Id, err)
		}
	}

	resp, err := NewAdminGetUserListLogic(context.Background(), svcCtx).AdminGetUserList(&types.AdminUserListReq{
		Page:     1,
		PageSize: 20,
		Status:   -1,
	})
	if err != nil {
		t.Fatalf("get user list: %v", err)
	}
	if !resp.Success || len(resp.List) != 3 {
		t.Fatalf("unexpected user list response: %#v", resp)
	}

	memberStatuses := map[int64]string{}
	memberExpiry := map[int64]string{}
	for _, item := range resp.List {
		memberStatuses[item.Id] = item.MemberStatus
		memberExpiry[item.Id] = item.MemberExpiresAt
	}

	if memberStatuses[1001] != "会员中" {
		t.Fatalf("expected active member status, got %#v", resp.List)
	}
	if memberExpiry[1001] == "" {
		t.Fatalf("expected active member expiry to be populated, got %#v", resp.List)
	}
	if memberStatuses[1002] != "已到期" {
		t.Fatalf("expected expired member status, got %#v", resp.List)
	}
	if memberStatuses[1003] != "未开通" {
		t.Fatalf("expected normal user status, got %#v", resp.List)
	}
}
