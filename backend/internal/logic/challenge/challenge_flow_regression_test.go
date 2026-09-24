package challenge

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/types"

	"gorm.io/gorm"
)

func TestEnterChallengeRejectsExpiryAfterChallengeLockWait(t *testing.T) {
	svcCtx := newChallengeLogicTestSvc(t)
	seedChallengeUsers(t, svcCtx, model.User{Id: 1001}, model.User{Id: 2002})
	challenge := seedAcceptedChallengeForEnter(t, svcCtx, 1001)
	if err := svcCtx.DB.Model(&model.Challenge{}).Where("id = ?", challenge.Id).
		Updates(map[string]any{"waiting_user_id": nil, "expires_at": time.Now().Add(300 * time.Millisecond)}).Error; err != nil {
		t.Fatal(err)
	}
	delayed := false
	if err := svcCtx.DB.Callback().Query().Before("gorm:query").Register("delay_challenge_lock", func(tx *gorm.DB) {
		if tx.Statement.Table == "challenges" && !delayed {
			if _, locked := tx.Statement.Clauses["FOR"]; locked {
				delayed = true
				time.Sleep(500 * time.Millisecond)
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	resp, err := NewEnterChallengeLogic(challengeTestCtx(1001), svcCtx).EnterChallenge(&types.EnterChallengeReq{ChallengeId: challenge.Id, ConfirmEarly: true})
	if err != nil || !delayed || resp.Success {
		t.Fatalf("过期前申请但在约球行锁后已到期，不能进入: delayed=%v resp=%+v err=%v", delayed, resp, err)
	}
}

func TestAcceptChallengeRejectsExpiryAfterChallengeLockWait(t *testing.T) {
	svcCtx := newChallengeLogicTestSvc(t)
	seedChallengeUsers(t, svcCtx, model.User{Id: 1001}, model.User{Id: 2002})
	challenge := seedAcceptedChallengeForEnter(t, svcCtx, 0)
	if err := svcCtx.DB.Model(&model.Challenge{}).Where("id = ?", challenge.Id).
		Updates(map[string]any{"status": model.ChallengeStatusPending, "waiting_user_id": nil, "expires_at": time.Now().Add(300 * time.Millisecond)}).Error; err != nil {
		t.Fatal(err)
	}
	delayed := false
	if err := svcCtx.DB.Callback().Query().Before("gorm:query").Register("delay_accept_lock", func(tx *gorm.DB) {
		if tx.Statement.Table == "challenges" && !delayed {
			if _, locked := tx.Statement.Clauses["FOR"]; locked {
				delayed = true
				time.Sleep(500 * time.Millisecond)
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	resp, err := NewAcceptChallengeLogic(challengeTestCtx(2002), svcCtx).AcceptChallenge(&types.AcceptChallengeReq{ChallengeId: challenge.Id})
	if err != nil || !delayed || resp.Success {
		t.Fatalf("约球行锁等待跨过截止时间后不得接受: delayed=%v resp=%+v err=%v", delayed, resp, err)
	}
}

func TestEnterChallengeReturnsOngoingMatchWithoutOpeningAnotherConnectionInTransaction(t *testing.T) {
	svcCtx := newChallengeLogicTestSvc(t)
	seedChallengeUsers(t, svcCtx, model.User{Id: 1001}, model.User{Id: 2002}, model.User{Id: 3003})
	challenge := seedAcceptedChallengeForEnter(t, svcCtx, 1001)
	other := int64(3003)
	if err := svcCtx.MatchModel.Create(&model.Match{Id: 81, UserId: 2002, OpponentId: &other, OpponentName: "C", GameType: 3, MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate, Status: 1, MatchTime: time.Now()}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(challengeTestCtx(2002), time.Second)
	defer cancel()
	resp, err := NewEnterChallengeLogic(ctx, svcCtx).EnterChallenge(&types.EnterChallengeReq{ChallengeId: challenge.Id})
	if err != nil || ctx.Err() != nil || !resp.Success || resp.Action != enterActionSelfOngoing || resp.MatchId != 81 || resp.OngoingMatch == nil {
		t.Fatalf("本人已有比赛应立即返回当前比赛，不得在事务中等待第二连接: resp=%+v err=%v ctx=%v", resp, err, ctx.Err())
	}
}

func TestExpiredChallengeHistoryIsEffectiveWithoutWorker(t *testing.T) {
	svcCtx := newChallengeLogicTestSvc(t)
	seedChallengeUsers(t, svcCtx, model.User{Id: 1001}, model.User{Id: 2002})
	challenge := seedAcceptedChallengeForEnter(t, svcCtx, 1001)
	if err := svcCtx.DB.Model(&model.Challenge{}).Where("id = ?", challenge.Id).Update("expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	resp, err := NewGetChallengeHistoryLogic(challengeTestCtx(1001), svcCtx).GetChallengeHistory(&types.GetChallengeHistoryReq{PageSize: 20})
	if err != nil || !resp.Success || len(resp.List) != 1 || resp.List[0].Status != model.ChallengeStatusExpired || resp.List[0].WaitingUserId != 0 {
		t.Fatalf("未执行 worker 时历史应显示已失效且无等待: resp=%+v err=%v", resp, err)
	}
}

func TestEnterChallengeResumesOngoingMatchBeforeFirstWait(t *testing.T) {
	svcCtx := newChallengeLogicTestSvc(t)
	seedChallengeUsers(t, svcCtx, model.User{Id: 1001}, model.User{Id: 2002}, model.User{Id: 3003})
	challenge := seedAcceptedChallengeForEnter(t, svcCtx, 1001)
	if _, err := svcCtx.ChallengeModel.ClearWaitingWithTx(nil, challenge.Id, 1001); err != nil {
		t.Fatal(err)
	}
	other := int64(3003)
	if err := svcCtx.MatchModel.Create(&model.Match{Id: 88, UserId: 1001, OpponentId: &other, OpponentName: "C", GameType: 3, MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate, Status: 1, MatchTime: time.Now()}); err != nil {
		t.Fatal(err)
	}
	resp, err := NewEnterChallengeLogic(challengeTestCtx(1001), svcCtx).EnterChallenge(&types.EnterChallengeReq{ChallengeId: challenge.Id, ConfirmEarly: true})
	if err != nil || resp.Action != enterActionSelfOngoing || resp.MatchId != 88 {
		t.Fatalf("首次进入时本人已有比赛应返回继续入口: resp=%+v err=%v", resp, err)
	}
	stored, _ := svcCtx.ChallengeModel.FindById(challenge.Id)
	if stored.WaitingUserId != nil {
		t.Fatalf("首次进入且本人已有比赛不得写入等待: %+v", stored)
	}
}

// commitFailingConnPool 注入提交失败：事务回滚后接口不得报告成功。
type commitFailingConnPool struct{ *sql.DB }

func (p commitFailingConnPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (gorm.ConnPool, error) {
	tx, err := p.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &commitFailingTx{tx}, nil
}

type commitFailingTx struct{ *sql.Tx }

func (tx *commitFailingTx) Commit() error {
	_ = tx.Tx.Rollback()
	return errors.New("injected commit failure")
}

func TestEnterChallengeDoesNotReportSuccessWhenCommitFails(t *testing.T) {
	svcCtx := newChallengeLogicTestSvc(t)
	seedChallengeUsers(t, svcCtx, model.User{Id: 1001}, model.User{Id: 2002})
	challenge := seedAcceptedChallengeForEnter(t, svcCtx, 1001)
	if _, err := svcCtx.ChallengeModel.ClearWaitingWithTx(nil, challenge.Id, 1001); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := svcCtx.DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	svcCtx.DB = svcCtx.DB.Session(&gorm.Session{NewDB: true})
	svcCtx.DB.Statement.ConnPool = commitFailingConnPool{sqlDB}
	resp, err := NewEnterChallengeLogic(challengeTestCtx(1001), svcCtx).EnterChallenge(&types.EnterChallengeReq{ChallengeId: challenge.Id, ConfirmEarly: true})
	if err != nil || resp.Success {
		t.Fatalf("提交失败回滚后不得报告等待成功: resp=%+v err=%v", resp, err)
	}
	stored, readErr := svcCtx.ChallengeModel.FindById(challenge.Id)
	if readErr != nil || stored.WaitingUserId != nil {
		t.Fatalf("等待必须被回滚: %+v err=%v", stored, readErr)
	}
}

func TestAcceptChallengeDoesNotReportExpiredAcceptedAsSuccess(t *testing.T) {
	svcCtx := newChallengeLogicTestSvc(t)
	seedChallengeUsers(t, svcCtx, model.User{Id: 1001}, model.User{Id: 2002})
	challenge := seedAcceptedChallengeForEnter(t, svcCtx, 1001)
	if err := svcCtx.DB.Model(&model.Challenge{}).Where("id = ?", challenge.Id).Update("expires_at", time.Now().Add(-time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	resp, err := NewAcceptChallengeLogic(challengeTestCtx(2002), svcCtx).AcceptChallenge(&types.AcceptChallengeReq{ChallengeId: challenge.Id})
	if err != nil || resp.Success || resp.Message != "约球已失效" {
		t.Fatalf("已失效的已接受记录重试不得报告成功: resp=%+v err=%v", resp, err)
	}
}

func TestEnterChallengeRejectsDisabledWaitingUser(t *testing.T) {
	svcCtx := newChallengeLogicTestSvc(t)
	seedChallengeUsers(t, svcCtx, model.User{Id: 1001}, model.User{Id: 2002})
	challenge := seedAcceptedChallengeForEnter(t, svcCtx, 1001)
	if err := svcCtx.UserModel.UpdateStatus(1001, 0); err != nil {
		t.Fatal(err)
	}
	resp, err := NewEnterChallengeLogic(challengeTestCtx(2002), svcCtx).EnterChallenge(&types.EnterChallengeReq{ChallengeId: challenge.Id})
	if err != nil || resp.Success || resp.MatchId > 0 {
		t.Fatalf("等待方账号被禁用后不得创建比赛: resp=%+v err=%v", resp, err)
	}
	stored, _ := svcCtx.ChallengeModel.FindById(challenge.Id)
	if stored.Status != model.ChallengeStatusAccepted {
		t.Fatalf("约球必须保持已接受: %+v", stored)
	}
}

func TestExpiredChallengeCannotBeCancelledOrAbandoned(t *testing.T) {
	for _, kind := range []string{"cancel", "abandon"} {
		t.Run(kind, func(t *testing.T) {
			svcCtx := newChallengeLogicTestSvc(t)
			seedChallengeUsers(t, svcCtx, model.User{Id: 1001}, model.User{Id: 2002})
			challenge := seedAcceptedChallengeForEnter(t, svcCtx, 1001)
			if err := svcCtx.DB.Model(&model.Challenge{}).Where("id = ?", challenge.Id).
				Update("expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
				t.Fatal(err)
			}
			var resp *types.ChallengeActionResp
			var err error
			if kind == "cancel" {
				resp, err = NewCancelChallengeLogic(challengeTestCtx(1001), svcCtx).CancelChallenge(&types.HandleChallengeReq{ChallengeId: challenge.Id})
			} else {
				resp, err = NewAbandonChallengeLogic(challengeTestCtx(2002), svcCtx).AbandonChallenge(&types.HandleChallengeReq{ChallengeId: challenge.Id})
			}
			if err != nil || resp.Success || resp.Message != "约球已失效" {
				t.Fatalf("已失效约球不得改写终态: kind=%s resp=%+v err=%v", kind, resp, err)
			}
			stored, _ := svcCtx.ChallengeModel.FindById(challenge.Id)
			if stored.Status != model.ChallengeStatusAccepted {
				t.Fatalf("存储状态不得被终态接口改写: %+v", stored)
			}
		})
	}
}

func TestSendChallengeRejectsSlotEndedDuringLockWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		now := logicx.NowUTC8()
		slotEnd := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, logicx.UTC8Location)
		// 推进虚拟时间到区间结束前 100ms，用户行锁等待 200ms 跨过区间结束。
		time.Sleep(slotEnd.Sub(now) - 100*time.Millisecond)
		svcCtx := newChallengeLogicTestSvc(t)
		seedChallengeUsers(t, svcCtx, model.User{Id: 1001}, model.User{Id: 2002})
		if err := svcCtx.DB.Create(&model.Friend{UserId: 1001, FriendId: 2002, Status: 1}).Error; err != nil {
			t.Fatal(err)
		}
		if err := svcCtx.DB.Create(&model.Friend{UserId: 2002, FriendId: 1001, Status: 1}).Error; err != nil {
			t.Fatal(err)
		}
		delayed := false
		if err := svcCtx.DB.Callback().Query().Before("gorm:query").Register("send_slot_end_wait", func(tx *gorm.DB) {
			if tx.Statement.Table == "users" && !delayed {
				if _, locked := tx.Statement.Clauses["FOR"]; locked {
					delayed = true
					time.Sleep(200 * time.Millisecond)
				}
			}
		}); err != nil {
			t.Fatal(err)
		}
		resp, err := NewSendChallengeLogic(challengeSendCtx(1001), svcCtx).SendChallenge(&types.SendChallengeReq{
			ToUserId: 2002, GameType: 3, MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate,
			MatchFormat: "free", DayOffset: 0, ScheduledDate: now.Format(time.DateOnly), StartHour: now.Hour(), EndHour: now.Hour() + 1,
		})
		if !delayed || !logicx.NowUTC8().After(slotEnd) {
			t.Fatalf("测试必须跨过区间结束时刻: delayed=%v now=%s end=%s", delayed, logicx.NowUTC8(), slotEnd)
		}
		if err != nil || resp.Success {
			t.Fatalf("锁等待跨过区间结束后不得发送: resp=%+v err=%v", resp, err)
		}
	})
}

func TestLeavingAlreadyStartedChallengeReturnsItsMatch(t *testing.T) {
	svcCtx := newChallengeLogicTestSvc(t)
	seedChallengeUsers(t, svcCtx, model.User{Id: 1001}, model.User{Id: 2002})
	challenge := seedAcceptedChallengeForEnter(t, svcCtx, 1001)
	started, err := NewEnterChallengeLogic(challengeTestCtx(2002), svcCtx).EnterChallenge(&types.EnterChallengeReq{ChallengeId: challenge.Id})
	if err != nil || !started.Success || started.MatchId <= 0 {
		t.Fatalf("开局失败: resp=%+v err=%v", started, err)
	}
	left, err := NewLeaveChallengeLogic(challengeTestCtx(1001), svcCtx).LeaveChallenge(&types.HandleChallengeReq{ChallengeId: challenge.Id})
	if err != nil || left.Success || left.MatchId != started.MatchId {
		t.Fatalf("开局后退出等待必须指向真实比赛: resp=%+v err=%v", left, err)
	}
}
