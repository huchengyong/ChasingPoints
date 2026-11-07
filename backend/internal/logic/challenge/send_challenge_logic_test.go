package challenge

import (
	"context"
	"testing"
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/types"

	"gorm.io/gorm"
)

func challengeSendCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func seedSendSchedule(t *testing.T) string {
	t.Helper()
	tomorrow := logicx.NowUTC8().AddDate(0, 0, 1)
	date := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, logicx.UTC8Location)
	return date.Format("2006-01-02")
}

func TestSendChallengeBlockedWhenEitherSideHasOpenJoinedChallenge(t *testing.T) {
	svcCtx := newChallengeLogicTestSvc(t)
	seedChallengeUsers(t, svcCtx,
		model.User{Id: 1001, Nickname: "小李"},
		model.User{Id: 2002, Nickname: "小王"},
		model.User{Id: 3003, Nickname: "小陈"},
	)
	challenge := seedAcceptedChallengeForEnter(t, svcCtx, 1001)
	// 让发送方与其他用户满足约球资格（好友关系）。
	for _, pair := range [][2]int64{{1001, 3003}, {3003, 2002}} {
		if err := svcCtx.DB.Create(&model.Friend{UserId: pair[0], FriendId: pair[1], Status: 1}).Error; err != nil {
			t.Fatalf("seed friendship: %v", err)
		}
		if err := svcCtx.DB.Create(&model.Friend{UserId: pair[1], FriendId: pair[0], Status: 1}).Error; err != nil {
			t.Fatalf("seed friendship: %v", err)
		}
	}
	scheduledDate := seedSendSchedule(t)

	send := func(sender, recipient int64) *types.SendChallengeResp {
		logic := NewSendChallengeLogic(challengeSendCtx(sender), svcCtx)
		resp, err := logic.SendChallenge(&types.SendChallengeReq{
			ToUserId:      int64(recipient),
			GameType:      3,
			MatchMode:     model.MatchModePractice,
			ScheduledDate: scheduledDate,
			StartHour:     16,
			EndHour:       17,
		})
		if err != nil {
			t.Fatalf("send challenge: %v", err)
		}
		return resp
	}

	// 发送方已有未结束约球：不能换人再发。
	if resp := send(1001, 3003); resp.Success || resp.Message != "你已有未结束的约球，请先处理当前约球" {
		t.Fatalf("sender with open challenge must be blocked: %#v", resp)
	}
	// 接收方已有未结束约球：发送仍允许（接受时才被资格校验拦截）。
	if resp := send(3003, 2002); !resp.Success || resp.ChallengeId <= 0 {
		t.Fatalf("send to busy recipient must be allowed: %#v", resp)
	}

	// 约球放弃后，双方都恢复资格。
	if _, err := svcCtx.ChallengeModel.UpdateStatusWithTx(nil, challenge.Id, []int{model.ChallengeStatusAccepted}, model.ChallengeStatusAbandoned,
		map[string]interface{}{"close_reason": model.ChallengeCloseReasonAbandoned}); err != nil {
		t.Fatalf("abandon challenge: %v", err)
	}
	resp := send(1001, 3003)
	if !resp.Success || resp.ChallengeId <= 0 {
		t.Fatalf("send must succeed after open challenge closed: %#v", resp)
	}
	if _, err := svcCtx.ChallengeModel.FindById(resp.ChallengeId); err != nil {
		t.Fatalf("challenge must persist: %v", err)
	}
}

func TestSendChallengeRechecksOccupancyAfterUserLock(t *testing.T) {
	svcCtx := newChallengeLogicTestSvc(t)
	seedChallengeUsers(t, svcCtx,
		model.User{Id: 1001, Nickname: "小李"},
		model.User{Id: 2002, Nickname: "小王"},
		model.User{Id: 3003, Nickname: "小陈"},
	)
	challenge := seedAcceptedChallengeForEnter(t, svcCtx, 1001)
	if err := svcCtx.DB.Model(&model.Challenge{}).Where("id = ?", challenge.Id).
		Update("expires_at", time.Now().Add(250*time.Millisecond)).Error; err != nil {
		t.Fatal(err)
	}
	if err := svcCtx.DB.Create(&model.Friend{UserId: 1001, FriendId: 3003, Status: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := svcCtx.DB.Create(&model.Friend{UserId: 3003, FriendId: 1001, Status: 1}).Error; err != nil {
		t.Fatal(err)
	}
	delayed := false
	if err := svcCtx.DB.Callback().Query().Before("gorm:query").Register("send_lock_wait", func(tx *gorm.DB) {
		if tx.Statement.Table == "users" && !delayed {
			if _, locked := tx.Statement.Clauses["FOR"]; locked {
				delayed = true
				time.Sleep(400 * time.Millisecond)
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	scheduledDate := seedSendSchedule(t)
	resp, err := NewSendChallengeLogic(challengeSendCtx(1001), svcCtx).SendChallenge(&types.SendChallengeReq{
		ToUserId: 3003, GameType: 3, MatchMode: model.MatchModePractice,
		ScheduledDate: scheduledDate, StartHour: 16, EndHour: 17,
	})
	if err != nil || !delayed || !resp.Success {
		t.Fatalf("用户锁等待跨过失效时刻后旧约球不得占用发送资格: resp=%+v err=%v delayed=%v", resp, err, delayed)
	}
}

func TestSendChallengeBlockedForStranger(t *testing.T) {
	svcCtx := newChallengeLogicTestSvc(t)
	seedChallengeUsers(t, svcCtx,
		model.User{Id: 1001, Nickname: "小李"},
		model.User{Id: 2002, Nickname: "陌生人"},
	)
	scheduledDate := seedSendSchedule(t)
	logic := NewSendChallengeLogic(challengeSendCtx(1001), svcCtx)
	resp, err := logic.SendChallenge(&types.SendChallengeReq{
		ToUserId:      2002,
		GameType:      3,
		MatchMode:     model.MatchModePractice,
		ScheduledDate: scheduledDate,
		StartHour:     16,
		EndHour:       17,
	})
	if err != nil {
		t.Fatalf("send challenge: %v", err)
	}
	if resp.Success || resp.Message == "" {
		t.Fatalf("stranger without shared match must be rejected: %#v", resp)
	}
}

func TestChallengeStartReqDerivesOptionsFromGameTypeAndMode(t *testing.T) {
	cases := []struct {
		name           string
		gameType       int
		mode           string
		wantMode       string
		wantVisibility string
		wantFormat     string
		wantSnooker    string
		wantVersion    int
		wantActor      int
	}{
		{name: "中八排位", gameType: 3, mode: model.MatchModeRanked, wantMode: model.MatchModeRanked, wantVisibility: model.MatchVisibilityPublic, wantFormat: model.MatchFormatFree, wantSnooker: model.SnookerFormatLegacy, wantVersion: model.SnookerRulesVersionLegacy},
		{name: "美九练习", gameType: 4, mode: model.MatchModePractice, wantMode: model.MatchModePractice, wantVisibility: model.MatchVisibilityPrivate, wantFormat: model.MatchFormatFree, wantSnooker: model.SnookerFormatLegacy, wantVersion: model.SnookerRulesVersionLegacy},
		{name: "九球追分沿用既有赛制", gameType: 2, mode: model.MatchModeRanked, wantMode: model.MatchModeRanked, wantVisibility: model.MatchVisibilityPublic, wantFormat: model.MatchFormatLegacy, wantSnooker: model.SnookerFormatLegacy, wantVersion: model.SnookerRulesVersionLegacy},
		{name: "斯诺克自由局数", gameType: 1, mode: "", wantMode: model.MatchModeRanked, wantVisibility: model.MatchVisibilityPublic, wantFormat: model.MatchFormatLegacy, wantSnooker: model.SnookerFormatFree, wantVersion: model.SnookerRulesVersionWPBSA, wantActor: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := challengeStartReq(1001, 2002, tc.gameType, tc.mode)
			if err != nil {
				t.Fatalf("challengeStartReq: %v", err)
			}
			if req.MatchMode != tc.wantMode || req.Visibility != tc.wantVisibility || req.MatchFormat != tc.wantFormat ||
				req.SnookerFormat != tc.wantSnooker || req.SnookerRulesVersion != tc.wantVersion || req.StartingActor != tc.wantActor {
				t.Fatalf("约球开局选项必须与扫码开局同一规范化结果: %+v", req)
			}
		})
	}
	for _, invalid := range []struct {
		gameType int
		mode     string
	}{{gameType: 0, mode: model.MatchModeRanked}, {gameType: 5, mode: model.MatchModeRanked}, {gameType: 3, mode: "casual"}} {
		if _, err := challengeStartReq(1001, 2002, invalid.gameType, invalid.mode); err == nil {
			t.Fatalf("invalid game type or mode must be rejected: %+v", invalid)
		}
	}
}
