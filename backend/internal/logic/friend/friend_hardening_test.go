package friend

import (
	"strings"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

func seedPendingFriendRequest(t *testing.T, svcCtx *svc.ServiceContext, fromUserId, toUserId int64, message string) *model.FriendRequest {
	t.Helper()

	request, err := svcCtx.FriendModel.SendRequest(fromUserId, toUserId, message)
	if err != nil {
		t.Fatalf("seed friend request: %v", err)
	}
	return request
}

func TestSendFriendRequestRejectsOverlongMessage(t *testing.T) {
	svcCtx := newFriendLogicTestSvc(t)
	seedFriendLogicUser(t, svcCtx, 101, "发起人")
	seedFriendLogicUser(t, svcCtx, 202, "目标用户")

	resp, err := NewSendFriendRequestLogic(friendLogicCtx(101), svcCtx).SendFriendRequest(&types.SendFriendRequestReq{
		ToUserId: 202,
		Message:  strings.Repeat("长", 201),
	})
	if err != nil {
		t.Fatalf("send friend request: %v", err)
	}
	if resp.Success || resp.Message != "留言不能超过 200 字" {
		t.Fatalf("expected overlong message rejection, got %#v", resp)
	}
	var count int64
	if err := svcCtx.DB.Model(&model.FriendRequest{}).Count(&count).Error; err != nil {
		t.Fatalf("count friend requests: %v", err)
	}
	if count != 0 {
		t.Fatalf("overlong message must not create request, got %d", count)
	}
}

func TestSendFriendRequestRejectedCooldownThenRetryAfterSevenDays(t *testing.T) {
	svcCtx := newFriendLogicTestSvc(t)
	seedFriendLogicUser(t, svcCtx, 101, "发起人")
	seedFriendLogicUser(t, svcCtx, 202, "目标用户")

	rejectedAt := time.Now().Add(-2 * 24 * time.Hour)
	if err := svcCtx.DB.Create(&model.FriendRequest{
		FromUserId: 101,
		ToUserId:   202,
		Status:     2,
		CreatedAt:  rejectedAt,
		UpdatedAt:  rejectedAt,
	}).Error; err != nil {
		t.Fatalf("seed rejected request: %v", err)
	}

	resp, err := NewSendFriendRequestLogic(friendLogicCtx(101), svcCtx).SendFriendRequest(&types.SendFriendRequestReq{ToUserId: 202})
	if err != nil {
		t.Fatalf("send friend request: %v", err)
	}
	if resp.Success || resp.Message != "近期已申请过，请过几天再试" {
		t.Fatalf("expected cooldown rejection, got %#v", resp)
	}

	if err := svcCtx.DB.Model(&model.FriendRequest{}).
		Where("from_user_id = ? AND to_user_id = ?", 101, 202).
		Update("updated_at", time.Now().Add(-8*24*time.Hour)).Error; err != nil {
		t.Fatalf("age rejected request: %v", err)
	}
	resp, err = NewSendFriendRequestLogic(friendLogicCtx(101), svcCtx).SendFriendRequest(&types.SendFriendRequestReq{ToUserId: 202})
	if err != nil {
		t.Fatalf("send friend request after cooldown: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected request allowed after 7 days, got %#v", resp)
	}
}

func TestSendFriendRequestDailyLimit(t *testing.T) {
	svcCtx := newFriendLogicTestSvc(t)
	seedFriendLogicUser(t, svcCtx, 101, "发起人")
	seedFriendLogicUser(t, svcCtx, 202, "目标用户")
	seedFriendLogicUser(t, svcCtx, 300, "今天的接收人")

	now := time.Now()
	requests := make([]model.FriendRequest, 0, friendRequestDailyLimit)
	for i := 0; i < friendRequestDailyLimit; i++ {
		requests = append(requests, model.FriendRequest{
			FromUserId: 101,
			ToUserId:   int64(1000 + i),
			Status:     2,
			CreatedAt:  now.Add(-time.Hour),
			UpdatedAt:  now.Add(-time.Hour),
		})
	}
	if err := svcCtx.DB.Create(&requests).Error; err != nil {
		t.Fatalf("seed daily requests: %v", err)
	}

	resp, err := NewSendFriendRequestLogic(friendLogicCtx(101), svcCtx).SendFriendRequest(&types.SendFriendRequestReq{ToUserId: 300})
	if err != nil {
		t.Fatalf("send friend request: %v", err)
	}
	if resp.Success || resp.Message != "今天发送的好友申请太多了，请明天再试" {
		t.Fatalf("expected daily limit rejection, got %#v", resp)
	}
}

func TestAcceptFriendRequestRejectsBlacklistRelation(t *testing.T) {
	svcCtx := newFriendLogicTestSvc(t)
	seedFriendLogicUser(t, svcCtx, 101, "接收人")
	seedFriendLogicUser(t, svcCtx, 202, "发起人")
	request := seedPendingFriendRequest(t, svcCtx, 202, 101, "加个好友")

	if err := svcCtx.DB.Create(&model.FriendBlacklist{UserId: 101, BlockedUserId: 202}).Error; err != nil {
		t.Fatalf("seed blacklist relation: %v", err)
	}
	resp, err := NewAcceptFriendRequestLogic(friendLogicCtx(101), svcCtx).AcceptFriendRequest(&types.HandleFriendRequestReq{RequestId: request.Id})
	if err != nil {
		t.Fatalf("accept friend request: %v", err)
	}
	if resp.Success || resp.Message != "由于隐私设置，无法添加对方为好友" {
		t.Fatalf("expected blacklist rejection, got %#v", resp)
	}
	var stored model.FriendRequest
	if err := svcCtx.DB.First(&stored, request.Id).Error; err != nil {
		t.Fatalf("load request: %v", err)
	}
	if stored.Status != 0 {
		t.Fatalf("blacklist rejection must keep request pending, got status %d", stored.Status)
	}
}

func TestAcceptFriendRequestRejectsWhenFriendLimitReached(t *testing.T) {
	svcCtx := newFriendLogicTestSvc(t)
	seedFriendLogicUser(t, svcCtx, 101, "接收人")
	seedFriendLogicUser(t, svcCtx, 202, "发起人")
	request := seedPendingFriendRequest(t, svcCtx, 202, 101, "加个好友")

	friends := make([]model.Friend, 0, friendLimit)
	for i := 0; i < friendLimit; i++ {
		friends = append(friends, model.Friend{UserId: 101, FriendId: int64(5000 + i), Status: 1})
	}
	if err := svcCtx.DB.Create(&friends).Error; err != nil {
		t.Fatalf("seed friends: %v", err)
	}

	resp, err := NewAcceptFriendRequestLogic(friendLogicCtx(101), svcCtx).AcceptFriendRequest(&types.HandleFriendRequestReq{RequestId: request.Id})
	if err != nil {
		t.Fatalf("accept friend request: %v", err)
	}
	if resp.Success || resp.Message != "好友数量已达上限" {
		t.Fatalf("expected friend limit rejection, got %#v", resp)
	}
	var stored model.FriendRequest
	if err := svcCtx.DB.First(&stored, request.Id).Error; err != nil {
		t.Fatalf("load request: %v", err)
	}
	if stored.Status != 0 {
		t.Fatalf("friend limit rejection must keep request pending, got status %d", stored.Status)
	}
}

func TestAcceptFriendRequestWritesFriendshipAndStatusAtomically(t *testing.T) {
	svcCtx := newFriendLogicTestSvc(t)
	seedFriendLogicUser(t, svcCtx, 101, "接收人")
	seedFriendLogicUser(t, svcCtx, 202, "发起人")
	request := seedPendingFriendRequest(t, svcCtx, 202, 101, "加个好友")

	resp, err := NewAcceptFriendRequestLogic(friendLogicCtx(101), svcCtx).AcceptFriendRequest(&types.HandleFriendRequestReq{RequestId: request.Id})
	if err != nil || !resp.Success {
		t.Fatalf("accept friend request: resp=%#v err=%v", resp, err)
	}
	var stored model.FriendRequest
	if err := svcCtx.DB.First(&stored, request.Id).Error; err != nil {
		t.Fatalf("load request: %v", err)
	}
	if stored.Status != 1 {
		t.Fatalf("expected accepted status, got %d", stored.Status)
	}
	areFriends, err := svcCtx.FriendModel.AreFriends(101, 202)
	if err != nil || !areFriends {
		t.Fatalf("expected bidirectional friendship: areFriends=%v err=%v", areFriends, err)
	}
}

func TestRejectFriendRequestRemovesPendingNotification(t *testing.T) {
	svcCtx := newFriendLogicTestSvc(t)
	seedFriendLogicUser(t, svcCtx, 101, "发起人")
	seedFriendLogicUser(t, svcCtx, 202, "接收人")

	resp, err := NewSendFriendRequestLogic(friendLogicCtx(101), svcCtx).SendFriendRequest(&types.SendFriendRequestReq{
		ToUserId: 202,
		Message:  "打两局吗",
	})
	if err != nil || !resp.Success {
		t.Fatalf("send friend request: resp=%#v err=%v", resp, err)
	}
	var request model.FriendRequest
	if err := svcCtx.DB.First(&request).Error; err != nil {
		t.Fatalf("load request: %v", err)
	}

	rejectResp, err := NewRejectFriendRequestLogic(friendLogicCtx(202), svcCtx).RejectFriendRequest(&types.HandleFriendRequestReq{RequestId: request.Id})
	if err != nil || !rejectResp.Success {
		t.Fatalf("reject friend request: resp=%#v err=%v", rejectResp, err)
	}
	var count int64
	if err := svcCtx.DB.Model(&model.Notification{}).Where("user_id = ? AND type = ?", 202, "friend_request").Count(&count).Error; err != nil {
		t.Fatalf("count notifications: %v", err)
	}
	if count != 0 {
		t.Fatalf("rejected request notification must be removed, got %d", count)
	}
}

func TestBlacklistListIncludesDeletedUsersAndUnblacklistRemovesRow(t *testing.T) {
	svcCtx := newFriendLogicTestSvc(t)
	seedFriendLogicUser(t, svcCtx, 101, "我")
	seedFriendLogicUser(t, svcCtx, 202, "被拉黑好友")
	seedFriendLogicUser(t, svcCtx, 303, "将被注销")

	logic := NewBlacklistFriendLogic(friendLogicCtx(101), svcCtx)
	if resp, err := logic.BlacklistFriend(&types.BlacklistFriendReq{FriendUserId: 202}); err != nil || !resp.Success {
		t.Fatalf("blacklist friend: resp=%#v err=%v", resp, err)
	}
	if resp, err := logic.BlacklistFriend(&types.BlacklistFriendReq{FriendUserId: 303}); err != nil || !resp.Success {
		t.Fatalf("blacklist second user: resp=%#v err=%v", resp, err)
	}
	if err := svcCtx.DB.Delete(&model.User{}, 303).Error; err != nil {
		t.Fatalf("soft delete user: %v", err)
	}

	listResp, err := NewGetFriendBlacklistLogic(friendLogicCtx(101), svcCtx).GetFriendBlacklist(&types.GetFriendBlacklistReq{Page: 1, PageSize: 20})
	if err != nil || !listResp.Success || listResp.Total != 2 || len(listResp.List) != 2 {
		t.Fatalf("get blacklist: resp=%#v err=%v", listResp, err)
	}
	if listResp.List[0].UserId != 303 || listResp.List[0].Nickname != model.DeletedUserDisplayName {
		t.Fatalf("deleted user must remain visible with deleted placeholder: %#v", listResp.List[0])
	}
	if listResp.List[1].UserId != 202 || listResp.List[1].Nickname != "被拉黑好友" {
		t.Fatalf("unexpected blacklist row: %#v", listResp.List[1])
	}

	unblacklistResp, err := NewUnblacklistFriendLogic(friendLogicCtx(101), svcCtx).UnblacklistFriend(&types.UnblacklistFriendReq{FriendUserId: 202})
	if err != nil || !unblacklistResp.Success || unblacklistResp.Message != "已移出黑名单" {
		t.Fatalf("unblacklist: resp=%#v err=%v", unblacklistResp, err)
	}
	hasRelation, err := svcCtx.FriendModel.HasBlacklistRelation(101, 202)
	if err != nil || hasRelation {
		t.Fatalf("expected blacklist row removed: hasRelation=%v err=%v", hasRelation, err)
	}
	missingResp, err := NewUnblacklistFriendLogic(friendLogicCtx(101), svcCtx).UnblacklistFriend(&types.UnblacklistFriendReq{FriendUserId: 202})
	if err != nil || missingResp.Success || missingResp.Message != "对方不在黑名单中" {
		t.Fatalf("unblacklist missing row: resp=%#v err=%v", missingResp, err)
	}
}

func TestBlacklistFriendClosesPendingChallengesOnly(t *testing.T) {
	svcCtx := newFriendLogicTestSvc(t)
	seedFriendLogicUser(t, svcCtx, 101, "我")
	seedFriendLogicUser(t, svcCtx, 202, "被拉黑人")

	now := time.Now()
	pending := &model.Challenge{
		FromUserId: 202,
		ToUserId:   101,
		GameType:   3,
		Status:     model.ChallengeStatusPending,
		ExpiresAt:  now.Add(time.Hour),
	}
	accepted := &model.Challenge{
		FromUserId: 101,
		ToUserId:   202,
		GameType:   3,
		Status:     model.ChallengeStatusAccepted,
		ExpiresAt:  now.Add(time.Hour),
	}
	for _, challenge := range []*model.Challenge{pending, accepted} {
		if err := svcCtx.DB.Create(challenge).Error; err != nil {
			t.Fatalf("seed challenge: %v", err)
		}
	}

	resp, err := NewBlacklistFriendLogic(friendLogicCtx(101), svcCtx).BlacklistFriend(&types.BlacklistFriendReq{FriendUserId: 202})
	if err != nil || !resp.Success {
		t.Fatalf("blacklist friend: resp=%#v err=%v", resp, err)
	}

	var storedPending model.Challenge
	if err := svcCtx.DB.First(&storedPending, pending.Id).Error; err != nil {
		t.Fatalf("load pending challenge: %v", err)
	}
	if storedPending.Status != model.ChallengeStatusCancelled || storedPending.CloseReason != model.ChallengeCloseReasonCancelled {
		t.Fatalf("pending challenge must be cancelled: %#v", storedPending)
	}
	var storedAccepted model.Challenge
	if err := svcCtx.DB.First(&storedAccepted, accepted.Id).Error; err != nil {
		t.Fatalf("load accepted challenge: %v", err)
	}
	if storedAccepted.Status != model.ChallengeStatusAccepted {
		t.Fatalf("accepted challenge must stay untouched, got status %d", storedAccepted.Status)
	}
}

func TestSendFriendRequestRejectsUnknownTarget(t *testing.T) {
	svcCtx := newFriendLogicTestSvc(t)
	seedFriendLogicUser(t, svcCtx, 101, "发起人")

	resp, err := NewSendFriendRequestLogic(friendLogicCtx(101), svcCtx).SendFriendRequest(&types.SendFriendRequestReq{ToUserId: 404404})
	if err != nil || resp.Success || resp.Message != "用户不存在" {
		t.Fatalf("unknown target: resp=%#v err=%v", resp, err)
	}
}
