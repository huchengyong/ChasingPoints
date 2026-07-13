package friend

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newFriendLogicTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := testsupport.PrepareFriendSchema(db); err != nil {
		t.Fatalf("prepare friend schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:                db,
		UserModel:         model.NewUserModel(db),
		FriendModel:       model.NewFriendModel(db),
		NotificationModel: model.NewNotificationModel(db),
	}
}

func seedFriendLogicUser(t *testing.T, svcCtx *svc.ServiceContext, id int64, nickname string) {
	t.Helper()

	if err := svcCtx.UserModel.Create(&model.User{
		Id:       id,
		Nickname: nickname,
		Status:   1,
	}); err != nil {
		t.Fatalf("create user %d: %v", id, err)
	}
}

func friendLogicCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func TestSendFriendRequestRejectsBlacklistedPair(t *testing.T) {
	svcCtx := newFriendLogicTestSvc(t)
	seedFriendLogicUser(t, svcCtx, 101, "发起人")
	seedFriendLogicUser(t, svcCtx, 202, "目标用户")

	if err := svcCtx.FriendModel.BlacklistFriend(202, 101); err != nil {
		t.Fatalf("seed blacklist relation: %v", err)
	}

	logic := NewSendFriendRequestLogic(friendLogicCtx(101), svcCtx)
	resp, err := logic.SendFriendRequest(&types.SendFriendRequestReq{
		ToUserId: 202,
		Message:  "想加个好友",
	})
	if err != nil {
		t.Fatalf("send friend request: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected request rejected, got %#v", resp)
	}
	if resp.Message != "由于隐私设置，无法发送好友申请" {
		t.Fatalf("unexpected response message: %#v", resp)
	}

	var requestCount int64
	if err := svcCtx.DB.Model(&model.FriendRequest{}).Count(&requestCount).Error; err != nil {
		t.Fatalf("count friend requests: %v", err)
	}
	if requestCount != 0 {
		t.Fatalf("expected 0 friend requests, got %d", requestCount)
	}
}

func TestSendFriendRequestCreatesNotificationWithRequestId(t *testing.T) {
	svcCtx := newFriendLogicTestSvc(t)
	seedFriendLogicUser(t, svcCtx, 101, "发起人")
	seedFriendLogicUser(t, svcCtx, 202, "目标用户")

	logic := NewSendFriendRequestLogic(friendLogicCtx(101), svcCtx)
	resp, err := logic.SendFriendRequest(&types.SendFriendRequestReq{
		ToUserId: 202,
		Message:  "一起开杆",
	})
	if err != nil {
		t.Fatalf("send friend request: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success response, got %#v", resp)
	}

	var request model.FriendRequest
	if err := svcCtx.DB.First(&request).Error; err != nil {
		t.Fatalf("find friend request: %v", err)
	}

	var notification model.Notification
	if err := svcCtx.DB.Where("user_id = ? AND type = ?", 202, "friend_request").First(&notification).Error; err != nil {
		t.Fatalf("find friend request notification: %v", err)
	}
	if notification.Data == nil {
		t.Fatal("expected notification payload")
	}
	if !strings.Contains(*notification.Data, fmt.Sprintf("\"request_id\":%d", request.Id)) {
		t.Fatalf("expected notification payload to include request id %d, got %s", request.Id, *notification.Data)
	}
}
