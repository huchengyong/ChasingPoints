package follow

import (
	"context"
	"testing"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newFollowLogicTestSvc(t *testing.T, socialEnabled bool) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := testsupport.PrepareFriendSchema(db); err != nil {
		t.Fatalf("prepare schema: %v", err)
	}
	if err := db.AutoMigrate(&model.Follow{}); err != nil {
		t.Fatalf("prepare follow schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:                db,
		UserModel:         model.NewUserModel(db),
		FriendModel:       model.NewFriendModel(db),
		FollowModel:       model.NewFollowModel(db),
		NotificationModel: model.NewNotificationModel(db),
		Config:            config.Config{Compliance: config.ComplianceConfig{RestrictedMode: true, AllowSocial: socialEnabled}},
	}
}

func seedFollowUser(t *testing.T, svcCtx *svc.ServiceContext, id int64, nickname string, status int) {
	t.Helper()
	if err := svcCtx.UserModel.Create(&model.User{Id: id, Nickname: nickname, Status: status}); err != nil {
		t.Fatalf("create user %d: %v", id, err)
	}
}

func followLogicCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func TestFollowEndpointsRejectedWhenSocialDisabled(t *testing.T) {
	svcCtx := newFollowLogicTestSvc(t, false)
	seedFollowUser(t, svcCtx, 101, "我", 1)
	seedFollowUser(t, svcCtx, 202, "对方", 1)

	wantMessage := config.DisabledFeatureMessage("social")
	followResp, err := NewFollowUserLogic(followLogicCtx(101), svcCtx).FollowUser(&types.FollowReq{TargetUserId: 202})
	if err != nil || followResp.Success || followResp.Message != wantMessage {
		t.Fatalf("follow must be disabled: resp=%#v err=%v", followResp, err)
	}
	unfollowResp, err := NewUnfollowUserLogic(followLogicCtx(101), svcCtx).UnfollowUser(&types.FollowReq{TargetUserId: 202})
	if err != nil || unfollowResp.Success || unfollowResp.Message != wantMessage {
		t.Fatalf("unfollow must be disabled: resp=%#v err=%v", unfollowResp, err)
	}
	followingResp, err := NewGetFollowingListLogic(followLogicCtx(101), svcCtx).GetFollowingList(&types.GetFollowListReq{Page: 1, PageSize: 20})
	if err != nil || followingResp.Success || followingResp.Message != wantMessage || len(followingResp.List) != 0 {
		t.Fatalf("following list must be disabled: resp=%#v err=%v", followingResp, err)
	}
	followerResp, err := NewGetFollowerListLogic(followLogicCtx(101), svcCtx).GetFollowerList(&types.GetFollowListReq{Page: 1, PageSize: 20})
	if err != nil || followerResp.Success || followerResp.Message != wantMessage || len(followerResp.List) != 0 {
		t.Fatalf("follower list must be disabled: resp=%#v err=%v", followerResp, err)
	}
}

func TestFollowUserNotifiesOnlyOnFirstFollow(t *testing.T) {
	svcCtx := newFollowLogicTestSvc(t, true)
	seedFollowUser(t, svcCtx, 101, "我", 1)
	seedFollowUser(t, svcCtx, 202, "对方", 1)

	for i := 0; i < 2; i++ {
		resp, err := NewFollowUserLogic(followLogicCtx(101), svcCtx).FollowUser(&types.FollowReq{TargetUserId: 202})
		if err != nil || !resp.Success {
			t.Fatalf("follow attempt %d: resp=%#v err=%v", i, resp, err)
		}
	}

	var count int64
	if err := svcCtx.DB.Model(&model.Notification{}).Where("user_id = ? AND type = ?", 202, "follow").Count(&count).Error; err != nil {
		t.Fatalf("count follow notifications: %v", err)
	}
	if count != 1 {
		t.Fatalf("repeated follow must not notify again, got %d notifications", count)
	}

	followingResp, err := NewGetFollowingListLogic(followLogicCtx(101), svcCtx).GetFollowingList(&types.GetFollowListReq{Page: 1, PageSize: 20})
	if err != nil || !followingResp.Success || len(followingResp.List) != 1 || followingResp.List[0].UserId != 202 || !followingResp.List[0].IsFollowing {
		t.Fatalf("following list: resp=%#v err=%v", followingResp, err)
	}
	followerResp, err := NewGetFollowerListLogic(followLogicCtx(202), svcCtx).GetFollowerList(&types.GetFollowListReq{Page: 1, PageSize: 20})
	if err != nil || !followerResp.Success || len(followerResp.List) != 1 || followerResp.List[0].UserId != 101 {
		t.Fatalf("follower list: resp=%#v err=%v", followerResp, err)
	}
}

func TestFollowUserRejectsMissingAndBlacklistedTargets(t *testing.T) {
	svcCtx := newFollowLogicTestSvc(t, true)
	seedFollowUser(t, svcCtx, 101, "我", 1)
	seedFollowUser(t, svcCtx, 202, "对方", 1)
	seedFollowUser(t, svcCtx, 303, "已停用", 2)

	missingResp, err := NewFollowUserLogic(followLogicCtx(101), svcCtx).FollowUser(&types.FollowReq{TargetUserId: 404404})
	if err != nil || missingResp.Success || missingResp.Message != "用户不存在" {
		t.Fatalf("missing target: resp=%#v err=%v", missingResp, err)
	}
	disabledResp, err := NewFollowUserLogic(followLogicCtx(101), svcCtx).FollowUser(&types.FollowReq{TargetUserId: 303})
	if err != nil || disabledResp.Success || disabledResp.Message != "用户不存在" {
		t.Fatalf("disabled target: resp=%#v err=%v", disabledResp, err)
	}

	if err := svcCtx.FriendModel.BlacklistFriend(202, 101); err != nil {
		t.Fatalf("seed blacklist relation: %v", err)
	}
	blacklistedResp, err := NewFollowUserLogic(followLogicCtx(101), svcCtx).FollowUser(&types.FollowReq{TargetUserId: 202})
	if err != nil || blacklistedResp.Success || blacklistedResp.Message != "由于隐私设置，无法关注对方" {
		t.Fatalf("blacklisted target: resp=%#v err=%v", blacklistedResp, err)
	}
}
