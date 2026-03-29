package friend

import (
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func TestBlacklistFriendRemovesFriendship(t *testing.T) {
	svcCtx := newFriendLogicTestSvc(t)
	seedFriendLogicUser(t, svcCtx, 101, "发起人")
	seedFriendLogicUser(t, svcCtx, 202, "目标用户")

	if err := svcCtx.FriendModel.AddFriend(101, 202); err != nil {
		t.Fatalf("seed friendship: %v", err)
	}

	logic := NewBlacklistFriendLogic(friendLogicCtx(101), svcCtx)
	resp, err := logic.BlacklistFriend(&types.BlacklistFriendReq{
		FriendUserId: 202,
	})
	if err != nil {
		t.Fatalf("blacklist friend: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success response, got %#v", resp)
	}

	areFriends, err := svcCtx.FriendModel.AreFriends(101, 202)
	if err != nil {
		t.Fatalf("check friendship: %v", err)
	}
	if areFriends {
		t.Fatal("expected friendship removed after blacklist")
	}

	hasBlacklistRelation, err := svcCtx.FriendModel.HasBlacklistRelation(101, 202)
	if err != nil {
		t.Fatalf("check blacklist relation: %v", err)
	}
	if !hasBlacklistRelation {
		t.Fatal("expected blacklist relation to exist")
	}
}

func TestBlacklistFriendRemovesPendingRequestNotificationsOnlyForBlockedUser(t *testing.T) {
	svcCtx := newFriendLogicTestSvc(t)
	seedFriendLogicUser(t, svcCtx, 101, "接收人")
	seedFriendLogicUser(t, svcCtx, 202, "待拉黑人")

	sendLogic := NewSendFriendRequestLogic(friendLogicCtx(202), svcCtx)
	resp, err := sendLogic.SendFriendRequest(&types.SendFriendRequestReq{
		ToUserId: 101,
		Message:  "打两局吗",
	})
	if err != nil {
		t.Fatalf("send friend request: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success response, got %#v", resp)
	}

	otherData := buildNotificationPayload("/subPages/social/friendRequests", 0, 999)
	if err := svcCtx.NotificationModel.Create(&model.Notification{
		UserId:  101,
		Type:    "friend_request",
		Title:   "收到好友申请",
		Content: "其他球友 向你发送了好友申请",
		Data:    otherData,
		IsRead:  0,
	}); err != nil {
		t.Fatalf("create unrelated notification: %v", err)
	}

	logic := NewBlacklistFriendLogic(friendLogicCtx(101), svcCtx)
	blacklistResp, err := logic.BlacklistFriend(&types.BlacklistFriendReq{
		FriendUserId: 202,
	})
	if err != nil {
		t.Fatalf("blacklist friend: %v", err)
	}
	if !blacklistResp.Success {
		t.Fatalf("expected success response, got %#v", blacklistResp)
	}

	var notifications []model.Notification
	if err := svcCtx.DB.Where("user_id = ? AND type = ?", 101, "friend_request").Order("id ASC").Find(&notifications).Error; err != nil {
		t.Fatalf("query notifications: %v", err)
	}
	if len(notifications) != 1 {
		t.Fatalf("expected 1 unrelated notification to remain, got %#v", notifications)
	}
	if notifications[0].Data == nil || *notifications[0].Data != *otherData {
		t.Fatalf("expected unrelated notification to remain untouched, got %#v", notifications[0])
	}
}
