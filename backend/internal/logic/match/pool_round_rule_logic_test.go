package match

import (
	"context"
	"fmt"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

// allowedWinTypes 的前两项为该球种的特殊成绩，其余为普胜。
type poolRoundGameCase struct {
	name            string
	gameType        int
	allowedWinTypes []string
	foreignWinType  string
}

var poolRoundGameCases = []poolRoundGameCase{
	{name: "eight-ball", gameType: 3, allowedWinTypes: []string{"break_clear", "continue_clear", "normal", ""}, foreignWinType: "big_gold"},
	{name: "american-nine-ball", gameType: 4, allowedWinTypes: []string{"small_gold", "big_gold", "normal", ""}, foreignWinType: "break_clear"},
}

var poolRoundFormats = []string{model.MatchFormatLegacy, model.MatchFormatFree}

type poolRoundMatchState struct {
	MyScore       int
	OpponentScore int
	Rounds        int64
	Actions       int64
	Revision      int64
}

func loadPoolRoundMatchState(t *testing.T, svcCtx *svc.ServiceContext, matchID int64) poolRoundMatchState {
	t.Helper()
	match, err := svcCtx.MatchModel.FindById(matchID)
	if err != nil || match == nil {
		t.Fatalf("reload match %d: %v", matchID, err)
	}
	rounds, err := svcCtx.MatchModel.GetRoundCount(matchID)
	if err != nil {
		t.Fatalf("count rounds: %v", err)
	}
	actions, err := svcCtx.MatchModel.CountActionsWithTx(nil, matchID)
	if err != nil {
		t.Fatalf("count actions: %v", err)
	}
	return poolRoundMatchState{
		MyScore:       match.MyScore,
		OpponentScore: match.OpponentScore,
		Rounds:        rounds,
		Actions:       actions,
		Revision:      match.SyncRevision,
	}
}

func endPoolRound(t *testing.T, svcCtx *svc.ServiceContext, matchID, userID int64, winner int, winType string, score int, clientActionID string) *types.EndRoundResp {
	t.Helper()
	match, err := svcCtx.MatchModel.FindById(matchID)
	if err != nil || match == nil {
		t.Fatalf("reload match %d: %v", matchID, err)
	}
	resp, err := NewEndRoundLogic(matchLogicCtx(userID), svcCtx).EndRound(&types.EndRoundReq{
		MatchId:        matchID,
		Winner:         winner,
		WinType:        winType,
		Score:          score,
		ClientActionId: clientActionID,
		BaseRevision:   match.SyncRevision,
	})
	if err != nil {
		t.Fatalf("end round %s: %v", clientActionID, err)
	}
	return resp
}

func TestPoolRoundEndRoundRejectsInvalidRounds(t *testing.T) {
	for _, game := range poolRoundGameCases {
		for _, format := range poolRoundFormats {
			t.Run(game.name+"/"+format, func(t *testing.T) {
				svcCtx := newPoolMatchFormatTestSvc(t)
				match := seedPoolFormatMatch(t, svcCtx, 801, func(match *model.Match) {
					match.GameType = game.gameType
					match.MatchFormat = format
				})
				rejected := []struct {
					name    string
					userID  int64
					winner  int
					winType string
					score   int
				}{
					{name: "player1 records own win", userID: 101, winner: 1, winType: "normal", score: 1},
					{name: "player2 records own win", userID: 202, winner: 2, winType: game.allowedWinTypes[0], score: 1},
					{name: "multi win round", userID: 101, winner: 2, winType: "normal", score: 2},
					{name: "zero win round", userID: 101, winner: 2, winType: "normal", score: 0},
					{name: "negative win round", userID: 101, winner: 2, winType: "normal", score: -50},
					{name: "start win type", userID: 101, winner: 2, winType: "start", score: 1},
					{name: "other pool win type", userID: 101, winner: 2, winType: game.foreignWinType, score: 1},
					{name: "snooker win type", userID: 101, winner: 2, winType: "break_147", score: 1},
				}
				before := loadPoolRoundMatchState(t, svcCtx, match.Id)
				for _, tc := range rejected {
					resp := endPoolRound(t, svcCtx, match.Id, tc.userID, tc.winner, tc.winType, tc.score, "rejected-"+tc.name)
					if resp.Success || resp.Accepted {
						t.Fatalf("%s: expected rejection, got %+v", tc.name, resp)
					}
					if after := loadPoolRoundMatchState(t, svcCtx, match.Id); after != before {
						t.Fatalf("%s: rejected round must not change state, before=%+v after=%+v", tc.name, before, after)
					}
				}
				achievements, err := svcCtx.MatchModel.GetAchievements(match.Id)
				if err != nil || len(achievements) != 0 {
					t.Fatalf("rejected rounds must not store achievements: %+v err=%v", achievements, err)
				}

				// 双方交替记录对手的胜局，覆盖本球种全部获胜方式（含不传 win_type）。
				for i, winType := range game.allowedWinTypes {
					userID, winner := int64(101), 2
					if i%2 == 1 {
						userID, winner = 202, 1
					}
					resp := endPoolRound(t, svcCtx, match.Id, userID, winner, winType, 1, fmt.Sprintf("accepted-%d", i))
					if !resp.Success || !resp.Accepted {
						t.Fatalf("round %d (%q) should be accepted: %+v", i, winType, resp)
					}
				}
				if after := loadPoolRoundMatchState(t, svcCtx, match.Id); after.MyScore != 2 || after.OpponentScore != 2 || after.Rounds != 4 {
					t.Fatalf("each accepted round must count exactly one win, got %+v", after)
				}
				achievements, err = svcCtx.MatchModel.GetAchievements(match.Id)
				if err != nil || len(achievements) != 2 {
					t.Fatalf("expected 2 special win records, got %+v err=%v", achievements, err)
				}
			})
		}
	}
}

func TestPoolRoundRefereeRecordsBothSides(t *testing.T) {
	for _, game := range poolRoundGameCases {
		t.Run(game.name, func(t *testing.T) {
			svcCtx := newPoolMatchFormatTestSvc(t)
			match := seedPoolFormatMatch(t, svcCtx, 802, func(match *model.Match) {
				match.GameType = game.gameType
				match.MatchFormat = model.MatchFormatLegacy
				match.RefereeUserId = int64Ref(303)
			})
			for _, winner := range []int{1, 2} {
				resp := endPoolRound(t, svcCtx, match.Id, 303, winner, "normal", 1, fmt.Sprintf("referee-%d", winner))
				if !resp.Success || !resp.Accepted {
					t.Fatalf("referee should record winner %d: %+v", winner, resp)
				}
			}
			if resp := endPoolRound(t, svcCtx, match.Id, 303, 1, "normal", 3, "referee-multi"); resp.Success || resp.Accepted {
				t.Fatalf("referee must not record a multi-win round either: %+v", resp)
			}
			if state := loadPoolRoundMatchState(t, svcCtx, match.Id); state.MyScore != 1 || state.OpponentScore != 1 || state.Rounds != 2 {
				t.Fatalf("expected 1:1 after referee rounds, got %+v", state)
			}
		})
	}
}

// win_type=start 的局会被局数统计忽略，一旦写入，撤销会误删上一局；拒绝后撤销只回退真实的局。
func TestPoolRoundStartWinTypeCannotCorruptUndo(t *testing.T) {
	for _, game := range poolRoundGameCases {
		t.Run(game.name, func(t *testing.T) {
			svcCtx := newPoolMatchFormatTestSvc(t)
			match := seedPoolFormatMatch(t, svcCtx, 803, func(match *model.Match) {
				match.GameType = game.gameType
			})
			if resp := endPoolRound(t, svcCtx, match.Id, 101, 2, "normal", 1, "round-1"); !resp.Success {
				t.Fatalf("first round failed: %+v", resp)
			}
			if resp := endPoolRound(t, svcCtx, match.Id, 101, 2, "start", 1, "round-start"); resp.Success {
				t.Fatalf("start win type must be rejected: %+v", resp)
			}
			state := loadPoolRoundMatchState(t, svcCtx, match.Id)
			undo, err := NewMatchUndoLogic(matchLogicCtx(101), svcCtx).MatchUndo(&types.MatchUndoReq{
				MatchId: match.Id, ClientActionId: "undo-1", BaseRevision: state.Revision,
			})
			if err != nil || !undo.Success {
				t.Fatalf("undo failed: %+v err=%v", undo, err)
			}
			var rounds int64
			if err := svcCtx.DB.Model(&model.MatchRound{}).Where("match_id = ?", match.Id).Count(&rounds).Error; err != nil {
				t.Fatalf("count raw rounds: %v", err)
			}
			if after := loadPoolRoundMatchState(t, svcCtx, match.Id); after.MyScore != 0 || after.OpponentScore != 0 || rounds != 0 {
				t.Fatalf("undo must remove the only round, got %+v raw rounds=%d", after, rounds)
			}
		})
	}
}

func TestPoolRoundRejectsScoreFoulAndNextRound(t *testing.T) {
	for _, game := range poolRoundGameCases {
		for _, format := range poolRoundFormats {
			t.Run(game.name+"/"+format, func(t *testing.T) {
				svcCtx := newPoolMatchFormatTestSvc(t)
				match := seedPoolFormatMatch(t, svcCtx, 804, func(match *model.Match) {
					match.GameType = game.gameType
					match.MatchFormat = format
				})
				before := loadPoolRoundMatchState(t, svcCtx, match.Id)

				for _, req := range []types.MatchScoreReq{
					{MatchId: match.Id, Actor: 1, Score: 5, ClientActionId: "score-positive"},
					{MatchId: match.Id, Actor: 2, Score: -30, ClientActionId: "score-negative"},
				} {
					resp, err := NewMatchScoreLogic(matchLogicCtx(101), svcCtx).MatchScore(&req)
					if err != nil || resp.Success || resp.Accepted || resp.Message != errPoolRoundScoreUnsupported.Error() {
						t.Fatalf("%s: direct score must be rejected: resp=%+v err=%v", req.ClientActionId, resp, err)
					}
				}
				foul, err := NewMatchFoulLogic(matchLogicCtx(101), svcCtx).MatchFoul(&types.MatchFoulReq{
					MatchId: match.Id, Actor: 1, Score: 40, ClientActionId: "foul",
				})
				if err != nil || foul.Success || foul.Accepted || foul.Message != errPoolRoundScoreUnsupported.Error() {
					t.Fatalf("foul must be rejected: resp=%+v err=%v", foul, err)
				}
				next, err := NewStartNextRoundLogic(matchLogicCtx(101), svcCtx).StartNextRound(&types.StartNextRoundReq{
					MatchId: match.Id, ClientActionId: "next-round",
				})
				if err != nil || next.Success || next.Accepted {
					t.Fatalf("next round must be rejected: resp=%+v err=%v", next, err)
				}
				if after := loadPoolRoundMatchState(t, svcCtx, match.Id); after != before {
					t.Fatalf("rejected writes must not change state, before=%+v after=%+v", before, after)
				}
			})
		}
	}
}

// 约球比赛由任一参赛方单方结束：自动达到抢N目标时与手动结束一致，直接结算而不进入双方确认。
func TestFlexiblePoolChallengeRankedAutoFinishEntersConfirmationAndCompletesOnTimeout(t *testing.T) {
	svcCtx := newPoolMatchFormatTestSvc(t)
	if err := svcCtx.DB.AutoMigrate(&model.Challenge{}); err != nil {
		t.Fatalf("prepare challenge schema: %v", err)
	}
	svcCtx.ChallengeModel = model.NewChallengeModel(svcCtx.DB)
	challengeID := int64(91)
	if err := svcCtx.ChallengeModel.Create(&model.Challenge{
		Id: challengeID, FromUserId: 101, ToUserId: 202,
		Status: model.ChallengeStatusStarted, ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("seed challenge: %v", err)
	}
	match := seedPoolFormatMatch(t, svcCtx, 807, func(match *model.Match) {
		match.MatchFormat = model.MatchFormatRaceTo
		match.TargetWins = 1
		match.MatchMode = model.MatchModeRanked
		match.ChallengeId = &challengeID
	})

	// 约球排位与扫码排位一致：达标自动进入30秒确认，发起方为记下最后一局的参赛者。
	resp := endPoolRound(t, svcCtx, match.Id, 101, 2, "normal", 1, "challenge-target-round")
	if !resp.Success || !resp.Accepted || resp.Snapshot.Status != 1 || resp.Snapshot.FinishState != model.FinishStatePendingConfirmation || resp.Snapshot.FinishRequestedBy != 101 {
		t.Fatalf("challenge ranked target should enter finish confirmation: %+v", resp.Snapshot)
	}
	if resp.Snapshot.FinishRequestRemainingSeconds != int(model.FinishRequestTTL.Seconds()) {
		t.Fatalf("fresh request should report the full countdown, got %d", resp.Snapshot.FinishRequestRemainingSeconds)
	}
	challenge, err := svcCtx.ChallengeModel.FindById(challengeID)
	if err != nil || challenge == nil || challenge.Status != model.ChallengeStatusStarted {
		t.Fatalf("challenge must stay started while confirming: %+v err=%v", challenge, err)
	}

	expiredAt := time.Now().Add(-model.FinishRequestTTL - time.Second)
	if err := svcCtx.DB.Model(&model.Match{}).Where("id = ?", match.Id).Update("finish_requested_at", expiredAt).Error; err != nil {
		t.Fatalf("age finish request: %v", err)
	}
	completion, err := completeExpiredFinishRequest(context.Background(), svcCtx, match.Id, time.Now())
	if err != nil || completion == nil {
		t.Fatalf("expired confirmation should complete the match: completion=%+v err=%v", completion, err)
	}
	stored, err := svcCtx.MatchModel.FindById(match.Id)
	if err != nil || stored.Status != 2 || stored.CompletionSource != model.CompletionSourceFinishTimeout {
		t.Fatalf("timeout must complete the challenge match: %+v err=%v", stored, err)
	}
	challenge, err = svcCtx.ChallengeModel.FindById(challengeID)
	if err != nil || challenge == nil || challenge.Status != model.ChallengeStatusCompleted {
		t.Fatalf("timeout must complete the linked challenge in the same transaction: %+v err=%v", challenge, err)
	}
}
