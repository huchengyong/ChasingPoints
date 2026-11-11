package challenge

import (
	"strings"
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

func seedChallengeFriendship(t *testing.T, svcCtx *svc.ServiceContext, a, b int64) {
	t.Helper()
	if err := svcCtx.FriendModel.AddFriend(a, b); err != nil {
		t.Fatalf("seed friendship: %v", err)
	}
}

func TestSendChallengeRejectsBlacklistedPair(t *testing.T) {
	svcCtx := newChallengeLogicTestSvc(t)
	seedChallengeUsers(t, svcCtx,
		model.User{Id: 1001, Nickname: "小李"},
		model.User{Id: 2002, Nickname: "小王"},
	)
	seedChallengeFriendship(t, svcCtx, 1001, 2002)
	if err := svcCtx.FriendModel.BlacklistFriend(2002, 1001); err != nil {
		t.Fatalf("seed blacklist relation: %v", err)
	}

	resp, err := NewSendChallengeLogic(challengeSendCtx(1001), svcCtx).SendChallenge(&types.SendChallengeReq{
		ToUserId:      2002,
		GameType:      3,
		MatchMode:     model.MatchModePractice,
		ScheduledDate: seedSendSchedule(t),
		StartHour:     19,
		EndHour:       20,
	})
	if err != nil {
		t.Fatalf("send challenge: %v", err)
	}
	if resp.Success || resp.Message != "由于隐私设置，无法向对方约球" {
		t.Fatalf("blacklisted pair must be rejected: %#v", resp)
	}
	var count int64
	if err := svcCtx.DB.Model(&model.Challenge{}).Count(&count).Error; err != nil {
		t.Fatalf("count challenges: %v", err)
	}
	if count != 0 {
		t.Fatalf("blacklisted pair must not create challenge, got %d", count)
	}
}

func TestSendChallengeRejectsOverlongMessage(t *testing.T) {
	svcCtx := newChallengeLogicTestSvc(t)
	seedChallengeUsers(t, svcCtx,
		model.User{Id: 1001, Nickname: "小李"},
		model.User{Id: 2002, Nickname: "小王"},
	)
	seedChallengeFriendship(t, svcCtx, 1001, 2002)

	resp, err := NewSendChallengeLogic(challengeSendCtx(1001), svcCtx).SendChallenge(&types.SendChallengeReq{
		ToUserId:      2002,
		GameType:      3,
		MatchMode:     model.MatchModePractice,
		Message:       strings.Repeat("长", 201),
		ScheduledDate: seedSendSchedule(t),
		StartHour:     19,
		EndHour:       20,
	})
	if err != nil {
		t.Fatalf("send challenge: %v", err)
	}
	if resp.Success || resp.Message != "留言不能超过 200 字" {
		t.Fatalf("overlong message must be rejected: %#v", resp)
	}
	var count int64
	if err := svcCtx.DB.Model(&model.Challenge{}).Count(&count).Error; err != nil {
		t.Fatalf("count challenges: %v", err)
	}
	if count != 0 {
		t.Fatalf("overlong message must not create challenge, got %d", count)
	}
}

func TestSendChallengeStoresTrimmedMessage(t *testing.T) {
	svcCtx := newChallengeLogicTestSvc(t)
	seedChallengeUsers(t, svcCtx,
		model.User{Id: 1001, Nickname: "小李"},
		model.User{Id: 2002, Nickname: "小王"},
	)
	seedChallengeFriendship(t, svcCtx, 1001, 2002)

	resp, err := NewSendChallengeLogic(challengeSendCtx(1001), svcCtx).SendChallenge(&types.SendChallengeReq{
		ToUserId:      2002,
		GameType:      3,
		MatchMode:     model.MatchModePractice,
		Message:       "  晚上来一局  ",
		ScheduledDate: seedSendSchedule(t),
		StartHour:     19,
		EndHour:       20,
	})
	if err != nil || !resp.Success {
		t.Fatalf("send challenge: resp=%#v err=%v", resp, err)
	}
	var challenge model.Challenge
	if err := svcCtx.DB.First(&challenge, resp.ChallengeId).Error; err != nil {
		t.Fatalf("load challenge: %v", err)
	}
	if challenge.Message != "晚上来一局" {
		t.Fatalf("expected trimmed message, got %q", challenge.Message)
	}
}
