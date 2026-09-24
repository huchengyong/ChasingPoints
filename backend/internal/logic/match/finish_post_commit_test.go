package match

import (
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg/ws"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/gorm"
)

// reviewV9PostCommitFixture 抢一练习赛约球比赛；automatic=true 时用 EndRound 触发自动结束，
// 否则预置一局后用 /finish 手动结束。
func reviewV9PostCommitFixture(t *testing.T, automatic bool) (*svc.ServiceContext, *model.Match, *ws.Hub) {
	t.Helper()
	svcCtx := newStartMatchChallengeTestSvc(t)
	if err := svcCtx.DB.AutoMigrate(&model.Notification{}); err != nil {
		t.Fatal(err)
	}
	svcCtx.NotificationModel = model.NewNotificationModel(svcCtx.DB)
	for _, id := range []int64{1001, 2002} {
		if err := svcCtx.UserModel.Create(&model.User{Id: id, Nickname: "player"}); err != nil {
			t.Fatal(err)
		}
	}
	opponent, challengeID := int64(2002), int64(77)
	if err := svcCtx.ChallengeModel.Create(&model.Challenge{
		Id: challengeID, FromUserId: 1001, ToUserId: opponent,
		Status: model.ChallengeStatusStarted, ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	m := &model.Match{
		Id: 88, UserId: 1001, OpponentId: &opponent, ChallengeId: &challengeID,
		GameType: 3, MatchFormat: model.MatchFormatFree, MyScore: 1, Status: 1, CurrentFrameStarted: true,
		MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate,
		FinishState: model.FinishStateNone, MatchTime: time.Now(),
	}
	if automatic {
		m.MatchFormat = model.MatchFormatRaceTo
		m.TargetWins = 1
		m.MyScore = 0
	}
	if err := svcCtx.MatchModel.Create(m); err != nil {
		t.Fatal(err)
	}
	if !automatic {
		winner := 1
		if err := svcCtx.DB.Create(&model.MatchRound{MatchId: m.Id, RoundNo: 1, Winner: &winner, WinType: "normal", MyScore: 1}).Error; err != nil {
			t.Fatal(err)
		}
	}
	old := ws.GlobalHub
	hub := ws.NewHub()
	ws.GlobalHub = hub
	t.Cleanup(func() { ws.GlobalHub = old })
	return svcCtx, m, hub
}

func reviewV9DrainMatchEnd(hub *ws.Hub) []map[string]interface{} {
	var ends []map[string]interface{}
	for len(hub.Broadcast) > 0 {
		message := <-hub.Broadcast
		var envelope struct {
			Type string                 `json:"type"`
			Data map[string]interface{} `json:"data"`
		}
		if json.Unmarshal(message.Message, &envelope) == nil && envelope.Type == "match_end" {
			ends = append(ends, envelope.Data)
		}
	}
	return ends
}

// 结束请求与真实记局并发使用同一幂等键：事务因 revision 冲突回滚后，
// 重放分支不得把别人的 win 动作当作已提交的结束，更不得广播未结束比赛的 match_end。
func TestFinishReplayConflictRejectsMismatchedAction(t *testing.T) {
	svcCtx, m, hub := reviewV9PostCommitFixture(t, false)
	injected := false
	key := "same-key-different-operation"
	if err := svcCtx.DB.Callback().Query().After("gorm:query").Register("review_v9_concurrent_win", func(tx *gorm.DB) {
		if injected || tx.Statement.Table != "match_actions" {
			return
		}
		injected = true
		response, err := NewEndRoundLogic(startReputationCtx(1001), svcCtx).EndRound(&types.EndRoundReq{
			MatchId: m.Id, Winner: 1, WinType: "normal", Score: 1, ClientActionId: key, BaseRevision: 0,
		})
		if err != nil || response == nil || !response.Success {
			t.Fatalf("concurrent round failed: %+v err=%v", response, err)
		}
	}); err != nil {
		t.Fatal(err)
	}
	response, err := NewFinishMatchLogic(startReputationCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId: m.Id, ClientActionId: key, BaseRevision: 0,
	})
	stored, readErr := svcCtx.MatchModel.FindById(m.Id)
	linked, linkErr := svcCtx.ChallengeModel.FindById(*m.ChallengeId)
	if err != nil || readErr != nil || linkErr != nil || !injected || stored == nil || linked == nil {
		t.Fatalf("fixture errors: err=%v/%v/%v injected=%v", err, readErr, linkErr, injected)
	}
	if got := len(reviewV9DrainMatchEnd(hub)); got > 0 {
		t.Errorf("revision conflict against a win action must not publish match_end: %d", got)
	}
	if response.Success || response.Accepted || stored.Status != 1 || linked.Status != model.ChallengeStatusStarted {
		t.Errorf("conflict must report failure and keep the match ongoing: response=%+v match=%+v challenge=%+v", response, stored, linked)
	}
}

// 自动结束提交后重读比赛失败：终态后处理必须用事务内已提交对象完成（失效事件/match_end 归属正确），响应不得假成功。
func TestAutoFinishReloadFailureUsesCommittedMatchForSideEffects(t *testing.T) {
	svcCtx, m, hub := reviewV9PostCommitFixture(t, true)
	matchesOutsideTx := 0
	failed := false
	if err := svcCtx.DB.Callback().Query().Before("gorm:query").Register("review_v9_match_reload_failure", func(tx *gorm.DB) {
		if _, inTx := tx.Statement.ConnPool.(*sql.Tx); !inTx && tx.Statement.Table == "matches" {
			matchesOutsideTx++
			if matchesOutsideTx == 2 {
				failed = true
				tx.AddError(errors.New("injected post-commit match reload failure"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	response, err := NewEndRoundLogic(startReputationCtx(1001), svcCtx).EndRound(&types.EndRoundReq{
		MatchId: m.Id, Winner: 1, WinType: "normal", Score: 1, ClientActionId: "v9-reload", BaseRevision: 0,
	})
	stored, readErr := svcCtx.MatchModel.FindById(m.Id)
	linked, linkErr := svcCtx.ChallengeModel.FindById(*m.ChallengeId)
	if err != nil || readErr != nil || linkErr != nil || !failed || stored == nil || stored.Status != 2 || linked == nil || linked.Status != model.ChallengeStatusCompleted {
		t.Fatalf("fixture not committed: match=%+v challenge=%+v err=%v/%v/%v failed=%v", stored, linked, err, readErr, linkErr, failed)
	}
	if got := drainChallengeInvalidations(hub); got != 2 {
		t.Errorf("committed challenge invalidations=%d want=2", got)
	}
	for _, end := range reviewV9DrainMatchEnd(hub) {
		if end["match_id"] != float64(m.Id) || end["status"] != float64(2) {
			t.Errorf("match_end must belong to the committed match: %+v", end)
		}
	}
	if response.Success {
		t.Errorf("reload failure must not return a fake success snapshot: %+v", response)
	}
}

// 自动结束后的快照组装失败：失效、赛果通知、成就同步与 match_end 都必须送达，响应保持成功且不出现零值快照。
func TestAutoFinishSideEffectsSurviveSnapshotAssemblyFailure(t *testing.T) {
	svcCtx, m, hub := reviewV9PostCommitFixture(t, true)
	failed := false
	if err := svcCtx.DB.Callback().Query().Before("gorm:query").Register("review_v9_postcommit_snapshot", func(tx *gorm.DB) {
		if _, inTx := tx.Statement.ConnPool.(*sql.Tx); !inTx && !failed && tx.Statement.Table == "match_rounds" {
			failed = true
			tx.AddError(errors.New("injected single post-commit snapshot failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	response, err := NewEndRoundLogic(startReputationCtx(1001), svcCtx).EndRound(&types.EndRoundReq{
		MatchId: m.Id, Winner: 1, WinType: "normal", Score: 1, ClientActionId: "v9-postcommit", BaseRevision: 0,
	})
	stored, readErr := svcCtx.MatchModel.FindById(m.Id)
	if err != nil || readErr != nil || !failed || stored == nil || stored.Status != 2 {
		t.Fatalf("fixture failed: resp=%+v err=%v/%v injected=%v", response, err, readErr, failed)
	}
	var notifications int64
	if err := svcCtx.DB.Model(&model.Notification{}).Where("type = ?", "match_result").Count(&notifications).Error; err != nil {
		t.Fatal(err)
	}
	ends := reviewV9DrainMatchEnd(hub)
	if invalidations := drainChallengeInvalidations(hub); invalidations != 2 {
		t.Errorf("challenge invalidations=%d want=2", invalidations)
	}
	if !response.Success || stored.AchievementSyncedAt == nil || notifications != 2 || len(ends) != 1 {
		t.Errorf("snapshot assembly failure must not skip side effects: response=%+v synced=%v notifications=%d ends=%d", response, stored.AchievementSyncedAt != nil, notifications, len(ends))
	}
	for _, end := range ends {
		if end["match_id"] != float64(m.Id) {
			t.Errorf("degraded match_end must carry authoritative match fields: %+v", end)
		}
	}
}

// 无故障对照组：同一场自动结束在无注入时完整走完，确保上述断言来自注入而非夹具本身。
func TestAutoFinishPostCommitHealthyControl(t *testing.T) {
	svcCtx, m, hub := reviewV9PostCommitFixture(t, true)
	response, err := NewEndRoundLogic(startReputationCtx(1001), svcCtx).EndRound(&types.EndRoundReq{
		MatchId: m.Id, Winner: 1, WinType: "normal", Score: 1, ClientActionId: "v9-control", BaseRevision: 0,
	})
	stored, readErr := svcCtx.MatchModel.FindById(m.Id)
	var notifications int64
	if err := svcCtx.DB.Model(&model.Notification{}).Where("type = ?", "match_result").Count(&notifications).Error; err != nil {
		t.Fatal(err)
	}
	ends := reviewV9DrainMatchEnd(hub)
	if err != nil || readErr != nil || !response.Success || stored == nil || stored.AchievementSyncedAt == nil || notifications != 2 || len(ends) != 1 {
		t.Fatalf("normal auto finish must complete the same fixture: response=%+v synced=%v notifications=%d ends=%d err=%v/%v", response, stored != nil && stored.AchievementSyncedAt != nil, notifications, len(ends), err, readErr)
	}
}
