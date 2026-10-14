package match

import (
	"encoding/json"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

const (
	nineBallTestMatchID  = int64(9201)
	nineBallTestPlayer1  = int64(1001)
	nineBallTestPlayer2  = int64(2002)
	nineBallTestGameType = 2
)

func seedNineBallTestMatch(t *testing.T, gameType int) *svc.ServiceContext {
	t.Helper()
	svcCtx := newMatchRefereeTestSvc(t)
	opponentID := nineBallTestPlayer2
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id:           nineBallTestMatchID,
		UserId:       nineBallTestPlayer1,
		OpponentId:   &opponentID,
		OpponentName: "选手乙",
		GameType:     gameType,
		MatchMode:    model.MatchModePractice,
		Visibility:   model.MatchVisibilityPrivate,
		Status:       1,
		MatchTime:    time.Now(),
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}
	return svcCtx
}

func reloadNineBallTestMatch(t *testing.T, svcCtx *svc.ServiceContext) *model.Match {
	t.Helper()
	match, err := svcCtx.MatchModel.FindById(nineBallTestMatchID)
	if err != nil || match == nil {
		t.Fatalf("reload match: %v", err)
	}
	return match
}

func endNineBallRound(t *testing.T, svcCtx *svc.ServiceContext, userID int64, req types.EndRoundReq) *types.EndRoundResp {
	t.Helper()
	req.MatchId = nineBallTestMatchID
	if req.ClientActionId == "" {
		req.ClientActionId = "end-" + req.WinType
	}
	req.BaseRevision = reloadNineBallTestMatch(t, svcCtx).SyncRevision
	resp, err := NewEndRoundLogic(matchRefereeCtx(userID), svcCtx).EndRound(&req)
	if err != nil {
		t.Fatalf("end round: %v", err)
	}
	return resp
}

func undoNineBallAction(t *testing.T, svcCtx *svc.ServiceContext, userID int64, clientActionID string) *types.MatchUndoResp {
	t.Helper()
	resp, err := NewMatchUndoLogic(matchRefereeCtx(userID), svcCtx).MatchUndo(&types.MatchUndoReq{
		MatchId:        nineBallTestMatchID,
		ClientActionId: clientActionID,
		BaseRevision:   reloadNineBallTestMatch(t, svcCtx).SyncRevision,
	})
	if err != nil {
		t.Fatalf("undo: %v", err)
	}
	return resp
}

func nineBallAchievementCount(t *testing.T, svcCtx *svc.ServiceContext, actor int, achievementType string) int {
	t.Helper()
	list, err := svcCtx.MatchModel.GetAchievements(nineBallTestMatchID)
	if err != nil {
		t.Fatalf("load achievements: %v", err)
	}
	for _, item := range list {
		if item.Actor == actor && item.AchievementType == achievementType {
			return item.Count
		}
	}
	return 0
}

func TestNineBallUndoWinRevertsScoreRoundAndAchievement(t *testing.T) {
	svcCtx := seedNineBallTestMatch(t, nineBallTestGameType)

	for _, id := range []string{"big-gold-1", "big-gold-2"} {
		resp := endNineBallRound(t, svcCtx, nineBallTestPlayer1, types.EndRoundReq{Winner: 2, WinType: "big_gold", Score: 10, ClientActionId: id})
		if !resp.Success {
			t.Fatalf("%s: expected success, got %+v", id, resp)
		}
	}
	rounds, err := svcCtx.MatchModel.ListCompletedRounds(nineBallTestMatchID)
	if err != nil || len(rounds) != 2 {
		t.Fatalf("expected 2 rounds, got %d err=%v", len(rounds), err)
	}
	if rounds[0].MyScore != 0 || rounds[0].OpponentScore != 10 || rounds[1].OpponentScore != 20 {
		t.Fatalf("局记录必须保存本局得分之后的累计比分: %+v", rounds)
	}
	if got := nineBallAchievementCount(t, svcCtx, 2, "nine_on_break"); got != 2 {
		t.Fatalf("expected 2 big gold records before undo, got %d", got)
	}

	if resp := undoNineBallAction(t, svcCtx, nineBallTestPlayer1, "undo-1"); !resp.Success {
		t.Fatalf("first undo failed: %+v", resp)
	}
	if got := nineBallAchievementCount(t, svcCtx, 2, "nine_on_break"); got != 1 {
		t.Fatalf("撤销一局大金后计数应回退为 1, got %d", got)
	}
	if match := reloadNineBallTestMatch(t, svcCtx); match.OpponentScore != 10 {
		t.Fatalf("expected opponent score 10 after first undo, got %d", match.OpponentScore)
	}

	if resp := undoNineBallAction(t, svcCtx, nineBallTestPlayer1, "undo-2"); !resp.Success {
		t.Fatalf("second undo failed: %+v", resp)
	}
	list, err := svcCtx.MatchModel.GetAchievements(nineBallTestMatchID)
	if err != nil || len(list) != 0 {
		t.Fatalf("撤销全部大金后不得残留特殊成绩记录: %+v err=%v", list, err)
	}
	rounds, _ = svcCtx.MatchModel.ListCompletedRounds(nineBallTestMatchID)
	match := reloadNineBallTestMatch(t, svcCtx)
	if len(rounds) != 0 || match.MyScore != 0 || match.OpponentScore != 0 {
		t.Fatalf("expected clean match after undo, rounds=%d score=%d:%d", len(rounds), match.MyScore, match.OpponentScore)
	}
}

func TestNineBallEndRoundEnforcesWinRules(t *testing.T) {
	rejected := []struct {
		name   string
		userID int64
		req    types.EndRoundReq
	}{
		{name: "invalid winner", userID: nineBallTestPlayer1, req: types.EndRoundReq{Winner: 7, WinType: "normal", Score: 4}},
		{name: "score does not match win type", userID: nineBallTestPlayer1, req: types.EndRoundReq{Winner: 2, WinType: "normal", Score: 999}},
		{name: "negative score", userID: nineBallTestPlayer1, req: types.EndRoundReq{Winner: 2, WinType: "normal", Score: -4}},
		{name: "unknown win type", userID: nineBallTestPlayer1, req: types.EndRoundReq{Winner: 2, WinType: `x","hack":"1`, Score: 4}},
		{name: "player1 credits self", userID: nineBallTestPlayer1, req: types.EndRoundReq{Winner: 1, WinType: "big_gold", Score: 10}},
		{name: "player2 credits self", userID: nineBallTestPlayer2, req: types.EndRoundReq{Winner: 2, WinType: "big_gold", Score: 10}},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			svcCtx := seedNineBallTestMatch(t, nineBallTestGameType)
			tc.req.ClientActionId = "rejected-round"
			if resp := endNineBallRound(t, svcCtx, tc.userID, tc.req); resp.Success {
				t.Fatalf("expected rejection, got %+v", resp)
			}
			match := reloadNineBallTestMatch(t, svcCtx)
			rounds, _ := svcCtx.MatchModel.ListCompletedRounds(nineBallTestMatchID)
			actionCount, _ := svcCtx.MatchModel.CountActionsWithTx(nil, nineBallTestMatchID)
			if match.MyScore != 0 || match.OpponentScore != 0 || len(rounds) != 0 || actionCount != 0 {
				t.Fatalf("rejected round must not write: score=%d:%d rounds=%d actions=%d", match.MyScore, match.OpponentScore, len(rounds), actionCount)
			}
		})
	}

	svcCtx := seedNineBallTestMatch(t, nineBallTestGameType)
	if resp := endNineBallRound(t, svcCtx, nineBallTestPlayer2, types.EndRoundReq{Winner: 1, WinType: "small_gold", Score: 7}); !resp.Success {
		t.Fatalf("player2 crediting player1 small gold must succeed: %+v", resp)
	}
	if match := reloadNineBallTestMatch(t, svcCtx); match.MyScore != 7 || match.OpponentScore != 0 {
		t.Fatalf("expected 7:0, got %d:%d", match.MyScore, match.OpponentScore)
	}
}

func TestNineBallFoulAndScoreEndpointsEnforceRules(t *testing.T) {
	svcCtx := seedNineBallTestMatch(t, nineBallTestGameType)
	foul := func(userID int64, actor int, score int, clientActionID string) *types.MatchScoreResp {
		resp, err := NewMatchFoulLogic(matchRefereeCtx(userID), svcCtx).MatchFoul(&types.MatchFoulReq{
			MatchId:        nineBallTestMatchID,
			Actor:          actor,
			Score:          score,
			ClientActionId: clientActionID,
			BaseRevision:   reloadNineBallTestMatch(t, svcCtx).SyncRevision,
		})
		if err != nil {
			t.Fatalf("foul: %v", err)
		}
		return resp
	}

	if resp := foul(nineBallTestPlayer1, 2, 5, "foul-score-5"); resp.Success || resp.Message != errNineBallInvalidFoulScore.Error() {
		t.Fatalf("foul score 5 must be rejected, got %+v", resp)
	}
	if resp := foul(nineBallTestPlayer1, 1, 0, "foul-self"); resp.Success || resp.Message != errNineBallOpponentOnly.Error() {
		t.Fatalf("player recording own foul must be rejected, got %+v", resp)
	}
	if resp := foul(nineBallTestPlayer1, 2, 0, "foul-opponent"); !resp.Success {
		t.Fatalf("opponent foul without score must succeed, got %+v", resp)
	}
	action, err := svcCtx.MatchModel.GetLastAction(nineBallTestMatchID)
	if err != nil || action == nil || action.ScoreChange != nineBallFoulScore {
		t.Fatalf("foul action must store fixed score, got %+v err=%v", action, err)
	}

	scoreResp, err := NewMatchScoreLogic(matchRefereeCtx(nineBallTestPlayer1), svcCtx).MatchScore(&types.MatchScoreReq{
		MatchId: nineBallTestMatchID, Actor: 1, Score: 50, ClientActionId: "direct-score", BaseRevision: reloadNineBallTestMatch(t, svcCtx).SyncRevision,
	})
	if err != nil {
		t.Fatalf("score: %v", err)
	}
	if scoreResp.Success || scoreResp.Message != errNineBallScoreUnsupported.Error() {
		t.Fatalf("nine-ball direct score must be rejected, got %+v", scoreResp)
	}
	if match := reloadNineBallTestMatch(t, svcCtx); match.MyScore != 1 || match.OpponentScore != 0 {
		t.Fatalf("expected only the opponent foul to count (1:0), got %d:%d", match.MyScore, match.OpponentScore)
	}
}

func TestNineBallUndoSkipsFinishFlowRecords(t *testing.T) {
	svcCtx := seedNineBallTestMatch(t, nineBallTestGameType)
	if resp := endNineBallRound(t, svcCtx, nineBallTestPlayer1, types.EndRoundReq{Winner: 2, WinType: "normal", Score: 4}); !resp.Success {
		t.Fatalf("end round failed: %+v", resp)
	}
	for _, actionType := range []string{"finish_request", "finish_dispute"} {
		if err := svcCtx.DB.Create(&model.MatchAction{MatchId: nineBallTestMatchID, ActionType: actionType, Actor: 1}).Error; err != nil {
			t.Fatalf("seed %s: %v", actionType, err)
		}
	}

	if resp := undoNineBallAction(t, svcCtx, nineBallTestPlayer1, "undo-after-dispute"); !resp.Success {
		t.Fatalf("undo failed: %+v", resp)
	}
	match := reloadNineBallTestMatch(t, svcCtx)
	rounds, _ := svcCtx.MatchModel.ListCompletedRounds(nineBallTestMatchID)
	if match.OpponentScore != 0 || len(rounds) != 0 {
		t.Fatalf("一次撤销必须直接回退上一局，got score=%d rounds=%d", match.OpponentScore, len(rounds))
	}
	var activeFinishRecords int64
	if err := svcCtx.DB.Model(&model.MatchAction{}).
		Where("match_id = ? AND action_type IN ? AND is_undone = 0", nineBallTestMatchID, []string{"finish_request", "finish_dispute"}).
		Count(&activeFinishRecords).Error; err != nil || activeFinishRecords != 2 {
		t.Fatalf("结束确认流程记录不得被撤销: count=%d err=%v", activeFinishRecords, err)
	}
}

func TestNineBallClientActionIDCannotBeReusedByAnotherAction(t *testing.T) {
	svcCtx := seedNineBallTestMatch(t, nineBallTestGameType)
	foulReq := types.MatchFoulReq{MatchId: nineBallTestMatchID, Actor: 2, ClientActionId: "shared-id", BaseRevision: 0}
	if resp, err := NewMatchFoulLogic(matchRefereeCtx(nineBallTestPlayer1), svcCtx).MatchFoul(&foulReq); err != nil || !resp.Success {
		t.Fatalf("foul failed: %+v err=%v", resp, err)
	}
	replay, err := NewMatchFoulLogic(matchRefereeCtx(nineBallTestPlayer1), svcCtx).MatchFoul(&foulReq)
	if err != nil || !replay.Success || !replay.Accepted {
		t.Fatalf("same request replay must succeed: %+v err=%v", replay, err)
	}

	resp := endNineBallRound(t, svcCtx, nineBallTestPlayer1, types.EndRoundReq{Winner: 2, WinType: "big_gold", Score: 10, ClientActionId: "shared-id"})
	if resp.Success || resp.Accepted {
		t.Fatalf("reused client_action_id for another action must be rejected, got %+v", resp)
	}
	match := reloadNineBallTestMatch(t, svcCtx)
	rounds, _ := svcCtx.MatchModel.ListCompletedRounds(nineBallTestMatchID)
	if match.MyScore != 1 || match.OpponentScore != 0 || len(rounds) != 0 {
		t.Fatalf("expected only the foul to count, got score=%d:%d rounds=%d", match.MyScore, match.OpponentScore, len(rounds))
	}
}

func TestLegacyEndRoundNonPositiveScoreCannotInflateOnUndo(t *testing.T) {
	// 中式八球、美式九球旧赛制同样走旧版结束一局与撤销路径：非正分值的局在入口即被拒绝，撤销无从凭空加分。
	for _, game := range []struct {
		name     string
		gameType int
	}{{name: "eight-ball", gameType: 3}, {name: "american-nine-ball", gameType: 4}} {
		t.Run(game.name, func(t *testing.T) {
			svcCtx := seedNineBallTestMatch(t, game.gameType)
			if resp := endNineBallRound(t, svcCtx, nineBallTestPlayer1, types.EndRoundReq{Winner: 2, WinType: "normal", Score: -50}); resp.Success || resp.Accepted {
				t.Fatalf("non-positive legacy round must be rejected: %+v", resp)
			}
			if resp := undoNineBallAction(t, svcCtx, nineBallTestPlayer1, "undo-negative"); resp.Success {
				t.Fatalf("rejected round must leave nothing to undo: %+v", resp)
			}
			if match := reloadNineBallTestMatch(t, svcCtx); match.MyScore != 0 || match.OpponentScore != 0 {
				t.Fatalf("撤销不得凭空加分, got %d:%d", match.MyScore, match.OpponentScore)
			}
		})
	}
}

func TestBuildWinActionExtraDataEscapesWinType(t *testing.T) {
	winType := `x","hack":"1`
	var payload map[string]any
	if err := json.Unmarshal([]byte(buildWinActionExtraData(winType)), &payload); err != nil {
		t.Fatalf("extra data must be valid JSON: %v", err)
	}
	if len(payload) != 1 || payload["win_type"] != winType {
		t.Fatalf("win_type must round-trip without injecting keys, got %+v", payload)
	}
}
