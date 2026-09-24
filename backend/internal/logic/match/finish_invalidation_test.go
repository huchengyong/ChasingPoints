package match

import (
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg/ws"
	"chasing_points/internal/types"

	"gorm.io/gorm"
)

// drainChallengeInvalidations 收集并清空 Hub 中待发的 challenge scope 失效事件。
func drainChallengeInvalidations(hub *ws.Hub) int {
	count := 0
	for len(hub.SendUser) > 0 {
		message := <-hub.SendUser
		var envelope struct {
			Type string `json:"type"`
			Data struct {
				Scopes []string `json:"scopes"`
			} `json:"data"`
		}
		if json.Unmarshal(message.Message, &envelope) != nil {
			continue
		}
		if envelope.Type == "user_data_updated" {
			for _, scope := range envelope.Data.Scopes {
				if scope == "challenge" {
					count++
					break
				}
			}
		}
	}
	return count
}

// 比赛终态事务提交后，失效通知必须送达双方：即使响应快照组装失败也不能漏发；
// 合法的幂等重放必须补发漏掉的事件。
func TestFinishInvalidationSurvivesPostCommitSnapshotFailure(t *testing.T) {
	svcCtx := newStartMatchChallengeTestSvc(t)
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
	match := &model.Match{
		Id: 88, UserId: 1001, OpponentId: &opponent, ChallengeId: &challengeID,
		GameType: 3, MatchFormat: "free", MyScore: 1, Status: 1,
		MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate,
		MatchTime: time.Now(),
	}
	if err := svcCtx.MatchModel.Create(match); err != nil {
		t.Fatal(err)
	}
	winner := 1
	if err := svcCtx.DB.Create(&model.MatchRound{MatchId: match.Id, RoundNo: 1, Winner: &winner, WinType: "normal", MyScore: 1}).Error; err != nil {
		t.Fatal(err)
	}
	oldHub := ws.GlobalHub
	hub := ws.NewHub()
	ws.GlobalHub = hub
	t.Cleanup(func() { ws.GlobalHub = oldHub })

	snapshotFailed := false
	if err := svcCtx.DB.Callback().Query().Before("gorm:query").Register("finish_snapshot_fail", func(tx *gorm.DB) {
		if !snapshotFailed && tx.Statement.Table == "match_rounds" {
			snapshotFailed = true
			tx.AddError(errors.New("injected snapshot read failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}

	req := &types.FinishMatchReq{MatchId: match.Id, ClientActionId: "invalidation-repair", BaseRevision: 0}
	logic := NewFinishMatchLogic(startReputationCtx(1001), svcCtx)
	first, err := logic.FinishMatch(req)

	stored, storedErr := svcCtx.MatchModel.FindById(match.Id)
	linked, linkedErr := svcCtx.ChallengeModel.FindById(challengeID)
	if storedErr != nil || linkedErr != nil || !snapshotFailed || stored.Status != 2 || linked.Status != model.ChallengeStatusCompleted {
		t.Fatalf("提交必须已完成并同步约球: match=%+v challenge=%+v err=%v/%v/%v", stored, linked, err, storedErr, linkedErr)
	}
	if got := drainChallengeInvalidations(hub); got != 2 {
		t.Fatalf("提交成功后双方都必须收到失效事件（快照失败也不能漏发）: events=%d response=%+v", got, first)
	}

	if err := svcCtx.DB.Callback().Query().Remove("finish_snapshot_fail"); err != nil {
		t.Fatal(err)
	}
	again, err := logic.FinishMatch(req)
	if err != nil || !again.Success || !again.Accepted {
		t.Fatalf("幂等重放必须成功: %+v err=%v", again, err)
	}
	if got := drainChallengeInvalidations(hub); got != 2 {
		t.Fatalf("合法重放必须补发失效事件: events=%d", got)
	}
}

// 灵活赛制自动结束提交后，响应快照读取失败也不能漏发约球失效通知。
func TestAutoFinishInvalidationSurvivesPostCommitSnapshotFailure(t *testing.T) {
	svcCtx := newStartMatchChallengeTestSvc(t)
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
	match := &model.Match{
		Id: 88, UserId: 1001, OpponentId: &opponent, ChallengeId: &challengeID,
		GameType: 3, MatchFormat: "race_to", TargetWins: 1, Status: 1, CurrentFrameStarted: true,
		MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate, MatchTime: time.Now(),
	}
	if err := svcCtx.MatchModel.Create(match); err != nil {
		t.Fatal(err)
	}
	oldHub := ws.GlobalHub
	hub := ws.NewHub()
	ws.GlobalHub = hub
	t.Cleanup(func() { ws.GlobalHub = oldHub })

	snapshotFailed := false
	if err := svcCtx.DB.Callback().Query().Before("gorm:query").Register("auto_finish_snapshot_fail", func(tx *gorm.DB) {
		// 事务内的合法读取不注入；只让提交后的响应快照读取失败。
		if _, inTx := tx.Statement.ConnPool.(*sql.Tx); !inTx && !snapshotFailed && tx.Statement.Table == "match_rounds" {
			snapshotFailed = true
			tx.AddError(errors.New("injected post-commit snapshot failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}

	resp, err := NewEndRoundLogic(startReputationCtx(1001), svcCtx).EndRound(&types.EndRoundReq{
		MatchId: match.Id, Winner: 1, Score: 1, WinType: "normal", ClientActionId: "auto-invalidation", BaseRevision: 0,
	})
	stored, storedErr := svcCtx.MatchModel.FindById(match.Id)
	linked, linkedErr := svcCtx.ChallengeModel.FindById(challengeID)
	if storedErr != nil || linkedErr != nil || !snapshotFailed || stored.Status != 2 || linked.Status != model.ChallengeStatusCompleted {
		t.Fatalf("自动结束必须提交终态: match=%+v challenge=%+v err=%v/%v/%v", stored, linked, err, storedErr, linkedErr)
	}
	if got := drainChallengeInvalidations(hub); got != 2 {
		t.Fatalf("自动结束后双方都必须收到失效事件（快照失败也不能漏发）: events=%d response=%+v", got, resp)
	}
}

// /finish 提交后的快照组装失败不得丢失 match_end：终态广播用权威字段降级补发，
// 响应不得出现假零值快照；幂等重放仍可再次广播。
func TestFinishSnapshotFailureDeliversDegradedMatchEnd(t *testing.T) {
	svcCtx := newStartMatchChallengeTestSvc(t)
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
	match := &model.Match{
		Id: 88, UserId: 1001, OpponentId: &opponent, ChallengeId: &challengeID,
		GameType: 3, MatchFormat: "free", MyScore: 1, Status: 1, CurrentFrameStarted: true,
		MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate, MatchTime: time.Now(),
	}
	if err := svcCtx.MatchModel.Create(match); err != nil {
		t.Fatal(err)
	}
	winner := 1
	if err := svcCtx.DB.Create(&model.MatchRound{MatchId: match.Id, RoundNo: 1, Winner: &winner, WinType: "normal", MyScore: 1}).Error; err != nil {
		t.Fatal(err)
	}
	oldHub := ws.GlobalHub
	hub := ws.NewHub()
	ws.GlobalHub = hub
	t.Cleanup(func() { ws.GlobalHub = oldHub })

	snapshotFailed := false
	if err := svcCtx.DB.Callback().Query().Before("gorm:query").Register("replay_match_end_fail", func(tx *gorm.DB) {
		if !snapshotFailed && tx.Statement.Table == "match_rounds" {
			snapshotFailed = true
			tx.AddError(errors.New("injected snapshot read failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}

	req := &types.FinishMatchReq{MatchId: match.Id, ClientActionId: "replay-match-end", BaseRevision: 0}
	logic := NewFinishMatchLogic(startReputationCtx(1001), svcCtx)
	first, err := logic.FinishMatch(req)
	if err != nil || !first.Success || !first.Accepted || !snapshotFailed {
		t.Fatalf("快照失败的首交必须以降级快照返回成功: %+v err=%v failed=%v", first, err, snapshotFailed)
	}
	if first.Snapshot.MatchId != match.Id || first.Snapshot.Status != 2 || first.Snapshot.ChallengeId != challengeID {
		t.Fatalf("降级快照必须携带权威比赛字段: %+v", first.Snapshot)
	}
	if got := drainChallengeInvalidations(hub); got != 2 {
		t.Fatalf("首交必须已发失效事件: events=%d", got)
	}
	drainMatchEnd := func() int {
		count := 0
		for len(hub.Broadcast) > 0 {
			message := <-hub.Broadcast
			var envelope struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(message.Message, &envelope) != nil {
				continue
			}
			if envelope.Type == "match_end" {
				count++
			}
		}
		return count
	}
	if got := drainMatchEnd(); got != 1 {
		t.Fatalf("快照失败的首交必须降级补发 match_end: events=%d", got)
	}
	if err := svcCtx.DB.Callback().Query().Remove("replay_match_end_fail"); err != nil {
		t.Fatal(err)
	}
	again, err := logic.FinishMatch(req)
	if err != nil || !again.Success || !again.Accepted {
		t.Fatalf("重放必须成功: %+v err=%v", again, err)
	}
	if got := drainMatchEnd(); got == 0 {
		t.Fatal("幂等重放必须再次广播 match_end")
	}
}
