package challenge

import (
	"context"
	"testing"
	"time"

	logicx "chasing_points/internal/logic"
	matchlogic "chasing_points/internal/logic/match"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func challengeTestCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func newChallengeLogicTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}, &model.Opponent{}, &model.Challenge{}, &model.MatchRound{}, &model.MatchAction{}, &model.Friend{}); err != nil {
		t.Fatalf("prepare challenge schema: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sqlite db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	return &svc.ServiceContext{
		DB:             db,
		UserModel:      model.NewUserModel(db),
		MatchModel:     model.NewMatchModel(db),
		FriendModel:    model.NewFriendModel(db),
		ChallengeModel: model.NewChallengeModel(db),
	}
}

func seedChallengeUsers(t *testing.T, svcCtx *svc.ServiceContext, users ...model.User) {
	t.Helper()
	for _, user := range users {
		u := user
		if err := svcCtx.UserModel.Create(&u); err != nil {
			t.Fatalf("create user %d: %v", user.Id, err)
		}
	}
}

func seedAcceptedChallengeForEnter(t *testing.T, svcCtx *svc.ServiceContext, waitingUser int64) *model.Challenge {
	t.Helper()
	scheduled := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	startHour, endHour := 16, 17
	challenge := &model.Challenge{
		Id:            9201,
		FromUserId:    1001,
		ToUserId:      2002,
		GameType:      3,
		ScheduledDate: &scheduled,
		StartHour:     &startHour,
		EndHour:       &endHour,
		MatchMode:     model.MatchModeRanked,
		Visibility:    model.MatchVisibilityPublic,
		MatchFormat:   "free",
		Status:        model.ChallengeStatusAccepted,
		WaitingUserId: challengeInt64Ptr(waitingUser),
		ExpiresAt:     time.Now().Add(12 * time.Hour),
	}
	if err := svcCtx.ChallengeModel.Create(challenge); err != nil {
		t.Fatalf("seed accepted challenge: %v", err)
	}
	return challenge
}

func TestEnterChallengeCreatesSingleMatchAndSingleFinishAllowed(t *testing.T) {
	svcCtx := newChallengeLogicTestSvc(t)
	seedChallengeUsers(t, svcCtx, model.User{Id: 1001, Nickname: "小李"}, model.User{Id: 2002, Nickname: "小王"})
	challenge := seedAcceptedChallengeForEnter(t, svcCtx, 1001)

	// B（第二人）进入 → 创建唯一比赛。
	resp, err := NewEnterChallengeLogic(challengeTestCtx(2002), svcCtx).EnterChallenge(&types.EnterChallengeReq{ChallengeId: challenge.Id})
	if err != nil || !resp.Success || resp.Action != enterActionMatchCreated || resp.MatchId <= 0 {
		t.Fatalf("second enter must create match: resp=%#v err=%v", resp, err)
	}
	created, err := svcCtx.MatchModel.FindById(resp.MatchId)
	if err != nil || created == nil {
		t.Fatalf("load created match: %v", err)
	}
	if created.ChallengeId == nil || *created.ChallengeId != challenge.Id {
		t.Fatalf("match must link challenge: %+v", created)
	}
	if created.FinishConfirmationRequired {
		t.Fatalf("challenge ranked match must allow single finish: %+v", created)
	}
	if created.MatchMode != model.MatchModeRanked || created.Visibility != model.MatchVisibilityPublic || created.MatchFormat != "free" {
		t.Fatalf("match options must come from invite: %+v", created)
	}

	// 重试/连点：返回同一场比赛，不重复创建。
	retry, err := NewEnterChallengeLogic(challengeTestCtx(2002), svcCtx).EnterChallenge(&types.EnterChallengeReq{ChallengeId: challenge.Id})
	if err != nil || !retry.Success || retry.MatchId != resp.MatchId {
		t.Fatalf("retry must resume same match: resp=%#v err=%v", retry, err)
	}
	ids, err := svcCtx.MatchModel.ListChallengeMatchIds([]int64{challenge.Id})
	if err != nil || len(ids) != 1 {
		t.Fatalf("exactly one match expected: ids=%v err=%v", ids, err)
	}

	// 完成一局后自由局数满足合法结束条件。
	winner := 1
	if _, err := matchlogic.NewEndRoundLogic(challengeTestCtx(1001), svcCtx).EndRound(&types.EndRoundReq{
		MatchId:        resp.MatchId,
		Winner:         winner,
		WinType:        "normal",
		Score:          1,
		ClientActionId: "round-1",
		BaseRevision:   0,
	}); err != nil {
		t.Fatalf("end round: %v", err)
	}

	// 参赛方 can_finish：约球排位单方可结束；非参与者不可见本局。
	for _, userID := range []int64{1001, 2002} {
		view, err := matchlogic.NewGetCurrentMatchLogic(challengeTestCtx(userID), svcCtx).GetCurrentMatch()
		if err != nil || view.Match == nil || !view.Match.CanFinish {
			t.Fatalf("player %d must be able to finish: view=%#v err=%v", userID, view, err)
		}
	}
}

func TestEnterChallengeFirstEnterOnlyWaits(t *testing.T) {
	svcCtx := newChallengeLogicTestSvc(t)
	seedChallengeUsers(t, svcCtx, model.User{Id: 1001, Nickname: "小李"}, model.User{Id: 2002, Nickname: "小王"})
	challenge := seedAcceptedChallengeForEnter(t, svcCtx, 1001)
	if _, err := svcCtx.ChallengeModel.ClearWaitingWithTx(nil, challenge.Id, 1001); err != nil {
		t.Fatalf("clear seeded waiting: %v", err)
	}

	resp, err := NewEnterChallengeLogic(challengeTestCtx(1001), svcCtx).EnterChallenge(&types.EnterChallengeReq{ChallengeId: challenge.Id})
	if err != nil || !resp.Success || resp.Action != enterActionWaiting || resp.MatchId != 0 {
		t.Fatalf("first enter must only wait: resp=%#v err=%v", resp, err)
	}
	stored, err := svcCtx.ChallengeModel.FindById(challenge.Id)
	if err != nil || stored.Status != model.ChallengeStatusAccepted || stored.WaitingUserId == nil || *stored.WaitingUserId != 1001 {
		t.Fatalf("challenge must record waiting user: %+v err=%v", stored, err)
	}

	// 退出等待后重新进入，仍只等待。
	if _, err := NewLeaveChallengeLogic(challengeTestCtx(1001), svcCtx).LeaveChallenge(&types.HandleChallengeReq{ChallengeId: challenge.Id}); err != nil {
		t.Fatalf("leave challenge: %v", err)
	}
	if stored, _ = svcCtx.ChallengeModel.FindById(challenge.Id); stored.WaitingUserId != nil {
		t.Fatalf("leave must clear waiting: %+v", stored)
	}
	if resp, err = NewEnterChallengeLogic(challengeTestCtx(1001), svcCtx).EnterChallenge(&types.EnterChallengeReq{ChallengeId: challenge.Id}); err != nil || resp.Action != enterActionWaiting {
		t.Fatalf("re-enter must wait again: resp=%#v err=%v", resp, err)
	}
}

func TestEnterChallengeEarlyRequiresConfirmationUntilStart(t *testing.T) {
	svcCtx := newChallengeLogicTestSvc(t)
	seedChallengeUsers(t, svcCtx, model.User{Id: 1001, Nickname: "小李"}, model.User{Id: 2002, Nickname: "小王"})
	challenge := seedAcceptedChallengeForEnter(t, svcCtx, 1001)
	if _, err := svcCtx.ChallengeModel.ClearWaitingWithTx(nil, challenge.Id, 1001); err != nil {
		t.Fatalf("clear seeded waiting: %v", err)
	}
	// 把预约日期改到明天，保证「未到开始时间」。
	tomorrow := logicx.NowUTC8().AddDate(0, 0, 1)
	tomorrow = time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, time.UTC)
	challenge.ScheduledDate = &tomorrow
	if err := svcCtx.DB.Model(&model.Challenge{}).Where("id = ?", challenge.Id).Update("scheduled_date", tomorrow).Error; err != nil {
		t.Fatalf("update schedule: %v", err)
	}

	resp, err := NewEnterChallengeLogic(challengeTestCtx(2002), svcCtx).EnterChallenge(&types.EnterChallengeReq{ChallengeId: challenge.Id})
	if err != nil || resp.Success || resp.Action != enterActionConfirmEarly {
		t.Fatalf("early enter without confirm must ask: resp=%#v err=%v", resp, err)
	}
	stored, _ := svcCtx.ChallengeModel.FindById(challenge.Id)
	if stored.WaitingUserId != nil {
		t.Fatalf("unconfirmed early enter must not write waiting: %+v", stored)
	}
	confirmed, err := NewEnterChallengeLogic(challengeTestCtx(2002), svcCtx).EnterChallenge(&types.EnterChallengeReq{ChallengeId: challenge.Id, ConfirmEarly: true})
	if err != nil || !confirmed.Success || confirmed.Action != enterActionWaiting {
		t.Fatalf("confirmed early enter must wait: resp=%#v err=%v", confirmed, err)
	}
}

func TestFinishChallengeMatchSettlesChallengeOnce(t *testing.T) {
	svcCtx := newChallengeLogicTestSvc(t)
	seedChallengeUsers(t, svcCtx, model.User{Id: 1001, Nickname: "小李"}, model.User{Id: 2002, Nickname: "小王"})
	challenge := seedAcceptedChallengeForEnter(t, svcCtx, 1001)
	// 用练习赛验证共享结算里的约球同步（排位结算依赖完整竞技模型，另有用例覆盖）。
	if err := svcCtx.DB.Model(&model.Challenge{}).Where("id = ?", challenge.Id).Updates(map[string]interface{}{
		"match_mode": model.MatchModePractice, "visibility": model.MatchVisibilityPrivate,
	}).Error; err != nil {
		t.Fatalf("switch to practice: %v", err)
	}

	resp, err := NewEnterChallengeLogic(challengeTestCtx(2002), svcCtx).EnterChallenge(&types.EnterChallengeReq{ChallengeId: challenge.Id})
	if err != nil || resp.MatchId <= 0 {
		t.Fatalf("enter to create match: resp=%#v err=%v", resp, err)
	}

	// 先完成一局（自由局数需至少一局），A 再单方结束。
	if _, err := matchlogic.NewEndRoundLogic(challengeTestCtx(1001), svcCtx).EndRound(&types.EndRoundReq{
		MatchId:        resp.MatchId,
		Winner:         1,
		WinType:        "normal",
		Score:          1,
		ClientActionId: "round-finish-1",
		BaseRevision:   0,
	}); err != nil {
		t.Fatalf("end round: %v", err)
	}
	finish, err := matchlogic.NewFinishMatchLogic(challengeTestCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId:        resp.MatchId,
		ClientActionId: "finish-challenge-1",
		BaseRevision:   1,
	})
	if err != nil || !finish.Success {
		t.Fatalf("single-sided finish must succeed: resp=%#v err=%v", finish, err)
	}
	stored, err := svcCtx.ChallengeModel.FindById(challenge.Id)
	if err != nil || stored.Status != model.ChallengeStatusCompleted || stored.CloseReason != model.ChallengeCloseReasonCompleted {
		t.Fatalf("challenge must be completed with match: %+v err=%v", stored, err)
	}
}

func TestCancelChallengeLosesToStartedMatch(t *testing.T) {
	svcCtx := newChallengeLogicTestSvc(t)
	seedChallengeUsers(t, svcCtx, model.User{Id: 1001, Nickname: "小李"}, model.User{Id: 2002, Nickname: "小王"})
	challenge := seedAcceptedChallengeForEnter(t, svcCtx, 1001)

	resp, err := NewEnterChallengeLogic(challengeTestCtx(2002), svcCtx).EnterChallenge(&types.EnterChallengeReq{ChallengeId: challenge.Id})
	if err != nil || resp.MatchId <= 0 {
		t.Fatalf("enter to create match: resp=%#v err=%v", resp, err)
	}

	cancelResp, err := NewCancelChallengeLogic(challengeTestCtx(1001), svcCtx).CancelChallenge(&types.HandleChallengeReq{ChallengeId: challenge.Id})
	if err != nil || cancelResp.Success || cancelResp.MatchId != resp.MatchId {
		t.Fatalf("cancel after start must fail with match id: resp=%#v err=%v", cancelResp, err)
	}
	stored, _ := svcCtx.ChallengeModel.FindById(challenge.Id)
	if stored.Status != model.ChallengeStatusStarted {
		t.Fatalf("challenge must stay started: %+v", stored)
	}
}

func TestAbandonEndsWaitingChallengeWithoutMatch(t *testing.T) {
	svcCtx := newChallengeLogicTestSvc(t)
	seedChallengeUsers(t, svcCtx, model.User{Id: 1001, Nickname: "小李"}, model.User{Id: 2002, Nickname: "小王"})
	challenge := seedAcceptedChallengeForEnter(t, svcCtx, 1001)

	abandon, err := NewAbandonChallengeLogic(challengeTestCtx(2002), svcCtx).AbandonChallenge(&types.HandleChallengeReq{ChallengeId: challenge.Id})
	if err != nil || !abandon.Success {
		t.Fatalf("abandon must succeed: resp=%#v err=%v", abandon, err)
	}
	stored, _ := svcCtx.ChallengeModel.FindById(challenge.Id)
	if stored.Status != model.ChallengeStatusAbandoned {
		t.Fatalf("challenge must be abandoned: %+v", stored)
	}
	if ids, _ := svcCtx.MatchModel.ListChallengeMatchIds([]int64{challenge.Id}); len(ids) != 0 {
		t.Fatalf("no match must exist: %v", ids)
	}
}

func challengeInt64Ptr(value int64) *int64 {
	copy := value
	return &copy
}

func TestEnterChallengeRejectsWhenExpiryCrossesDuringLockWait(t *testing.T) {
	svcCtx := newChallengeLogicTestSvc(t)
	seedChallengeUsers(t, svcCtx, model.User{Id: 1001, Nickname: "小李"}, model.User{Id: 2002, Nickname: "小王"})
	challenge := seedAcceptedChallengeForEnter(t, svcCtx, 1001)
	// 失效时刻将近：用户行锁的查询被人为延迟，模拟临界排队跨过次晨07:00。
	expires := time.Now().Add(300 * time.Millisecond)
	if err := svcCtx.DB.Model(&model.Challenge{}).Where("id = ?", challenge.Id).
		Updates(map[string]interface{}{"waiting_user_id": nil, "expires_at": expires}).Error; err != nil {
		t.Fatalf("tighten expiry: %v", err)
	}
	delayed := false
	if err := svcCtx.DB.Callback().Query().Before("gorm:query").Register("review_delay_user_lock", func(tx *gorm.DB) {
		if tx.Statement.Table == "users" && !delayed {
			delayed = true
			time.Sleep(500 * time.Millisecond)
		}
	}); err != nil {
		t.Fatalf("register delay callback: %v", err)
	}

	resp, err := NewEnterChallengeLogic(challengeTestCtx(1001), svcCtx).EnterChallenge(&types.EnterChallengeReq{
		ChallengeId: challenge.Id, ConfirmEarly: true,
	})
	if err != nil {
		t.Fatalf("enter challenge: %v", err)
	}
	if !delayed || !time.Now().After(expires) {
		t.Fatalf("test must cross expiry during lock wait: delayed=%v", delayed)
	}
	if resp.Success {
		t.Fatalf("entry must be rejected after expiry crossed during lock wait: %#v", resp)
	}
}

func TestChallengeSummaryCarriesWaitingAndSchedule(t *testing.T) {
	svcCtx := newChallengeLogicTestSvc(t)
	seedChallengeUsers(t, svcCtx, model.User{Id: 1001, Nickname: "小李"}, model.User{Id: 2002, Nickname: "小王"})
	challenge := seedAcceptedChallengeForEnter(t, svcCtx, 1001)

	resp, err := NewGetChallengeSummaryLogic(challengeTestCtx(2002), svcCtx).GetChallengeSummary()
	if err != nil || !resp.Success || resp.CurrentChallenge == nil {
		t.Fatalf("summary: resp=%#v err=%v", resp, err)
	}
	got := resp.CurrentChallenge
	if got.WaitingUserId != 1001 || got.StartHour != *challenge.StartHour || got.EndHour != *challenge.EndHour {
		t.Fatalf("summary lost waiting/schedule: waiting=%d start=%d end=%d", got.WaitingUserId, got.StartHour, got.EndHour)
	}
}

func TestChallengeDetailDerivesExpiredWithoutWorker(t *testing.T) {
	svcCtx := newChallengeLogicTestSvc(t)
	seedChallengeUsers(t, svcCtx, model.User{Id: 1001, Nickname: "小李"}, model.User{Id: 2002, Nickname: "小王"})
	challenge := seedAcceptedChallengeForEnter(t, svcCtx, 1001)
	if err := svcCtx.DB.Model(&model.Challenge{}).Where("id = ?", challenge.Id).
		Update("expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatalf("expire challenge: %v", err)
	}

	resp, err := NewGetChallengeDetailLogic(challengeTestCtx(1001), svcCtx).GetChallengeDetail(&types.GetChallengeDetailReq{Id: challenge.Id})
	if err != nil || !resp.Success || resp.Challenge == nil {
		t.Fatalf("detail: resp=%#v err=%v", resp, err)
	}
	if resp.Challenge.Status != model.ChallengeStatusExpired {
		t.Fatalf("expired accepted detail must be derived as expired: status=%d", resp.Challenge.Status)
	}
}
