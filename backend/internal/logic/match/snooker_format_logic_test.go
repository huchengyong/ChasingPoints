package match

import (
	"fmt"
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func seedFlexibleSnookerMatch(t *testing.T, id int64, format string, targetWins int, mode string) (*model.Match, *UpdateSnookerFormatLogic) {
	t.Helper()
	svcCtx := newSnookerActionTestSvc(t)
	match := seedSnookerActionMatch(t, svcCtx, id, 0, 0)
	match.SnookerFormat = format
	match.SnookerTargetWins = targetWins
	match.BestOfFrames = 0
	match.MatchMode = mode
	match.FinishConfirmationRequired = mode == model.MatchModeRanked
	if mode == model.MatchModeRanked {
		match.Visibility = model.MatchVisibilityPublic
	}
	if err := svcCtx.DB.Save(match).Error; err != nil {
		t.Fatalf("update flexible match: %v", err)
	}
	return match, NewUpdateSnookerFormatLogic(matchLogicCtx(101), svcCtx)
}

func TestUpdateSnookerFormatBeforeScoring(t *testing.T) {
	match, logic := seedFlexibleSnookerMatch(t, 1010, model.SnookerFormatFree, 0, model.MatchModePractice)
	resp, err := logic.UpdateSnookerFormat(&types.UpdateSnookerFormatReq{
		MatchId: match.Id, SnookerFormat: model.SnookerFormatRaceTo, SnookerTargetWins: 10, BaseRevision: 0,
	})
	if err != nil || !resp.Success || !resp.Accepted || resp.ServerRevision != 1 || resp.Snapshot.SnookerFormat != model.SnookerFormatRaceTo || resp.Snapshot.SnookerTargetWins != 10 || !resp.Snapshot.CanChangeSnookerFormat {
		t.Fatalf("update format failed: resp=%+v err=%v", resp, err)
	}

	opponentResp, err := NewUpdateSnookerFormatLogic(matchLogicCtx(202), logic.svcCtx).UpdateSnookerFormat(&types.UpdateSnookerFormatReq{
		MatchId: match.Id, SnookerFormat: model.SnookerFormatFree, BaseRevision: 1,
	})
	if err != nil || opponentResp.Success || opponentResp.Accepted || opponentResp.ServerRevision != 1 {
		t.Fatalf("opponent update should be rejected: resp=%+v err=%v", opponentResp, err)
	}

	conflict, err := logic.UpdateSnookerFormat(&types.UpdateSnookerFormatReq{
		MatchId: match.Id, SnookerFormat: model.SnookerFormatFree, BaseRevision: 0,
	})
	if err != nil || conflict.Success || conflict.Accepted || conflict.ServerRevision != 1 {
		t.Fatalf("stale revision should be rejected: resp=%+v err=%v", conflict, err)
	}
}

func TestUpdateSnookerFormatLocksAfterProgress(t *testing.T) {
	t.Run("action", func(t *testing.T) {
		match, logic := seedFlexibleSnookerMatch(t, 1020, model.SnookerFormatFree, 0, model.MatchModePractice)
		if err := logic.svcCtx.DB.Create(&model.MatchAction{MatchId: match.Id, RoundNo: 1, ActionType: model.MatchActionTypeSnookerStroke, Actor: 1}).Error; err != nil {
			t.Fatalf("seed action: %v", err)
		}
		resp, _ := logic.UpdateSnookerFormat(&types.UpdateSnookerFormatReq{MatchId: match.Id, SnookerFormat: model.SnookerFormatRaceTo, SnookerTargetWins: 3, BaseRevision: 0})
		if resp.Success || resp.Accepted || resp.Snapshot.CanChangeSnookerFormat {
			t.Fatalf("progressed match should lock format: %+v", resp)
		}
	})

	t.Run("completed round", func(t *testing.T) {
		match, logic := seedFlexibleSnookerMatch(t, 1021, model.SnookerFormatFree, 0, model.MatchModePractice)
		winner := 1
		if err := logic.svcCtx.DB.Create(&model.MatchRound{MatchId: match.Id, RoundNo: 1, Winner: &winner}).Error; err != nil {
			t.Fatalf("seed round: %v", err)
		}
		resp, _ := logic.UpdateSnookerFormat(&types.UpdateSnookerFormatReq{MatchId: match.Id, SnookerFormat: model.SnookerFormatRaceTo, SnookerTargetWins: 3, BaseRevision: 0})
		if resp.Success || resp.Accepted || resp.Snapshot.CanChangeSnookerFormat {
			t.Fatalf("completed round should lock format: %+v", resp)
		}
	})

	t.Run("referee bound", func(t *testing.T) {
		match, logic := seedFlexibleSnookerMatch(t, 1022, model.SnookerFormatFree, 0, model.MatchModePractice)
		refereeID := int64(303)
		match.RefereeUserId = &refereeID
		if err := logic.svcCtx.DB.Save(match).Error; err != nil {
			t.Fatalf("seed referee: %v", err)
		}
		resp, _ := logic.UpdateSnookerFormat(&types.UpdateSnookerFormatReq{MatchId: match.Id, SnookerFormat: model.SnookerFormatRaceTo, SnookerTargetWins: 3, BaseRevision: 0})
		if resp.Success || resp.Accepted || resp.Snapshot.CanChangeSnookerFormat {
			t.Fatalf("referee-bound match should lock format: %+v", resp)
		}
	})

	t.Run("finish confirmation", func(t *testing.T) {
		match, logic := seedFlexibleSnookerMatch(t, 1023, model.SnookerFormatFree, 0, model.MatchModeRanked)
		match.FinishState = model.FinishStatePendingConfirmation
		if err := logic.svcCtx.DB.Save(match).Error; err != nil {
			t.Fatalf("seed finish state: %v", err)
		}
		resp, _ := logic.UpdateSnookerFormat(&types.UpdateSnookerFormatReq{MatchId: match.Id, SnookerFormat: model.SnookerFormatRaceTo, SnookerTargetWins: 3, BaseRevision: 0})
		if resp.Success || resp.Accepted || resp.Snapshot.CanChangeSnookerFormat {
			t.Fatalf("pending finish match should lock format: %+v", resp)
		}
	})
}

func TestFreeSnookerNormalFinishAllowsLeadAndTie(t *testing.T) {
	for _, tt := range []struct {
		name          string
		id            int64
		myScore       int
		opponentScore int
		wantResult    int
	}{
		{name: "lead", id: 1030, myScore: 2, opponentScore: 1, wantResult: 1},
		{name: "tie", id: 1031, myScore: 1, opponentScore: 1, wantResult: 3},
	} {
		t.Run(tt.name, func(t *testing.T) {
			match, logic := seedFlexibleSnookerMatch(t, tt.id, model.SnookerFormatFree, 0, model.MatchModePractice)
			match.MyScore = tt.myScore
			match.OpponentScore = tt.opponentScore
			match.CurrentFrameStarted = false
			if err := logic.svcCtx.DB.Save(match).Error; err != nil {
				t.Fatalf("seed score: %v", err)
			}
			resp, err := NewFinishMatchLogic(matchLogicCtx(101), logic.svcCtx).FinishMatch(&types.FinishMatchReq{
				MatchId: match.Id, ClientActionId: "finish-" + tt.name, BaseRevision: 0,
			})
			if err != nil || !resp.Success || !resp.Accepted || resp.Result != tt.wantResult || resp.Snapshot.Status != 2 {
				t.Fatalf("normal finish failed: resp=%+v err=%v", resp, err)
			}
		})
	}
}

func TestFreeSnookerNormalFinishReplayIsIdempotent(t *testing.T) {
	match, logic := seedFlexibleSnookerMatch(t, 1032, model.SnookerFormatFree, 0, model.MatchModePractice)
	match.MyScore = 1
	match.CurrentFrameStarted = false
	if err := logic.svcCtx.DB.Save(match).Error; err != nil {
		t.Fatalf("seed score: %v", err)
	}
	req := &types.FinishMatchReq{MatchId: match.Id, ClientActionId: "finish-idempotent", BaseRevision: 0}
	first, err := NewFinishMatchLogic(matchLogicCtx(101), logic.svcCtx).FinishMatch(req)
	if err != nil || !first.Success || !first.Accepted || first.Snapshot.Status != 2 {
		t.Fatalf("first finish failed: resp=%+v err=%v", first, err)
	}
	second, err := NewFinishMatchLogic(matchLogicCtx(101), logic.svcCtx).FinishMatch(req)
	if err != nil || !second.Success || !second.Accepted || second.Snapshot.Status != 2 || second.ServerRevision != first.ServerRevision {
		t.Fatalf("finish replay should return completed snapshot: first=%+v second=%+v err=%v", first, second, err)
	}
}

func TestFreeSnookerRejectsZeroRoundAndInFrameFinish(t *testing.T) {
	for _, tt := range []struct {
		name                string
		id                  int64
		myScore             int
		currentFrameStarted bool
	}{
		{name: "zero round", id: 1040, currentFrameStarted: false},
		{name: "in frame", id: 1041, myScore: 1, currentFrameStarted: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			match, logic := seedFlexibleSnookerMatch(t, tt.id, model.SnookerFormatFree, 0, model.MatchModePractice)
			match.MyScore = tt.myScore
			match.CurrentFrameStarted = tt.currentFrameStarted
			if err := logic.svcCtx.DB.Save(match).Error; err != nil {
				t.Fatalf("seed state: %v", err)
			}
			resp, err := NewFinishMatchLogic(matchLogicCtx(101), logic.svcCtx).FinishMatch(&types.FinishMatchReq{
				MatchId: match.Id, ClientActionId: "reject-" + tt.name, BaseRevision: 0,
			})
			if err != nil || resp.Success || resp.Accepted || resp.Snapshot.Status != 1 {
				t.Fatalf("finish should be rejected: resp=%+v err=%v", resp, err)
			}
		})
	}
}

func TestFreeSnookerRankedFinishUsesConfirmation(t *testing.T) {
	match, logic := seedFlexibleSnookerMatch(t, 1050, model.SnookerFormatFree, 0, model.MatchModeRanked)
	match.MyScore = 1
	match.OpponentScore = 1
	match.CurrentFrameStarted = false
	if err := logic.svcCtx.DB.Save(match).Error; err != nil {
		t.Fatalf("seed ranked state: %v", err)
	}
	resp, err := NewRequestFinishMatchLogic(matchLogicCtx(101), logic.svcCtx).RequestFinishMatch(&types.FinishMatchActionReq{
		MatchId: match.Id, ClientActionId: "ranked-free-request", BaseRevision: 0,
	})
	if err != nil || !resp.Success || !resp.Accepted || resp.FinishState != model.FinishStatePendingConfirmation || resp.Snapshot.Status != 1 {
		t.Fatalf("ranked free finish should wait for confirmation: resp=%+v err=%v", resp, err)
	}
}

func TestRaceToSnookerRejectsEarlyNormalFinish(t *testing.T) {
	match, logic := seedFlexibleSnookerMatch(t, 1051, model.SnookerFormatRaceTo, 3, model.MatchModePractice)
	match.MyScore = 1
	match.OpponentScore = 1
	match.CurrentFrameStarted = false
	if err := logic.svcCtx.DB.Save(match).Error; err != nil {
		t.Fatalf("seed race-to state: %v", err)
	}
	resp, err := NewFinishMatchLogic(matchLogicCtx(101), logic.svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId: match.Id, ClientActionId: "race-to-early-finish", BaseRevision: 0,
	})
	if err != nil || resp.Success || resp.Accepted || resp.Snapshot.Status != 1 {
		t.Fatalf("race-to match should reject early normal finish: resp=%+v err=%v", resp, err)
	}
}

func TestMatchConcessionUsesFlexibleFormatScoreWithoutSecondConfirmation(t *testing.T) {
	for _, tt := range []struct {
		name            string
		id              int64
		format          string
		targetWins      int
		mode            string
		wantWinnerScore int
	}{
		{name: "practice free", id: 1052, format: model.SnookerFormatFree, mode: model.MatchModePractice, wantWinnerScore: 1},
		{name: "practice race to", id: 1053, format: model.SnookerFormatRaceTo, targetWins: 3, mode: model.MatchModePractice, wantWinnerScore: 3},
	} {
		t.Run(tt.name, func(t *testing.T) {
			match, formatLogic := seedFlexibleSnookerMatch(t, tt.id, tt.format, tt.targetWins, tt.mode)
			offer, err := NewSnookerFrameActionLogic(matchLogicCtx(101), formatLogic.svcCtx).SnookerFrameAction(&types.SnookerFrameActionReq{
				MatchId: match.Id, Actor: 1, Action: model.SnookerFrameActionOfferConcession, Scope: model.SnookerConcessionScopeMatch,
				ClientActionId: fmt.Sprintf("offer-%d", match.Id), BaseRevision: 0,
			})
			if err != nil || !offer.Success || !offer.Accepted {
				t.Fatalf("offer match concession failed: resp=%+v err=%v", offer, err)
			}
			accept, err := NewSnookerFrameActionLogic(matchLogicCtx(202), formatLogic.svcCtx).SnookerFrameAction(&types.SnookerFrameActionReq{
				MatchId: match.Id, Actor: 2, Action: model.SnookerFrameActionAcceptConcession, Scope: model.SnookerConcessionScopeMatch,
				ClientActionId: fmt.Sprintf("accept-%d", match.Id), BaseRevision: offer.ServerRevision,
			})
			if err != nil || !accept.Success || !accept.Accepted || accept.Snapshot.Status != 2 || accept.Snapshot.FinishState != model.FinishStateNone || accept.Snapshot.MyScore != tt.wantWinnerScore {
				t.Fatalf("accept match concession failed: resp=%+v err=%v", accept, err)
			}
		})
	}
}

func TestRankedMatchConcessionDoesNotRequireSecondFinishConfirmation(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	seedFinishReputationUsers(t, svcCtx,
		model.User{Id: 101, Nickname: "选手1", Status: 1},
		model.User{Id: 202, Nickname: "选手2", Status: 1},
	)
	opponentID := int64(202)
	match := &model.Match{
		Id:                         1054,
		UserId:                     101,
		OpponentId:                 &opponentID,
		OpponentName:               "选手2",
		GameType:                   1,
		SnookerRulesVersion:        model.SnookerRulesVersionWPBSA,
		SnookerFormat:              model.SnookerFormatFree,
		StartingActor:              1,
		MatchMode:                  model.MatchModeRanked,
		Visibility:                 model.MatchVisibilityPublic,
		FinishConfirmationRequired: true,
		CurrentFrameStarted:        true,
		Status:                     1,
	}
	if err := svcCtx.MatchModel.Create(match); err != nil {
		t.Fatalf("seed ranked snooker match: %v", err)
	}
	offer, err := NewSnookerFrameActionLogic(matchLogicCtx(101), svcCtx).SnookerFrameAction(&types.SnookerFrameActionReq{
		MatchId: match.Id, Actor: 1, Action: model.SnookerFrameActionOfferConcession, Scope: model.SnookerConcessionScopeMatch,
		ClientActionId: "ranked-offer-1054", BaseRevision: 0,
	})
	if err != nil || !offer.Success || !offer.Accepted {
		t.Fatalf("offer ranked match concession failed: resp=%+v err=%v", offer, err)
	}
	accept, err := NewSnookerFrameActionLogic(matchLogicCtx(202), svcCtx).SnookerFrameAction(&types.SnookerFrameActionReq{
		MatchId: match.Id, Actor: 2, Action: model.SnookerFrameActionAcceptConcession, Scope: model.SnookerConcessionScopeMatch,
		ClientActionId: "ranked-accept-1054", BaseRevision: offer.ServerRevision,
	})
	if err != nil || !accept.Success || !accept.Accepted || accept.Snapshot.Status != 2 || accept.Snapshot.FinishState != model.FinishStateNone || accept.Snapshot.MyScore != 1 || accept.Snapshot.CanConfirmFinish {
		t.Fatalf("ranked concession should settle immediately after acceptance: resp=%+v err=%v", accept, err)
	}
}

func TestSnookerFormatLimits(t *testing.T) {
	for _, target := range []int{1, 10, 25} {
		t.Run(fmt.Sprintf("race to %d", target), func(t *testing.T) {
			match := &model.Match{GameType: 1, SnookerRulesVersion: model.SnookerRulesVersionWPBSA, SnookerFormat: model.SnookerFormatRaceTo, SnookerTargetWins: target, MyScore: target}
			if !snookerFormatLimitReached(match) || !model.SnookerTargetReached(match) {
				t.Fatalf("target %d should be reached", target)
			}
		})
	}
	free := &model.Match{GameType: 1, SnookerRulesVersion: model.SnookerRulesVersionWPBSA, SnookerFormat: model.SnookerFormatFree, MyScore: 25, OpponentScore: 24}
	if !snookerFormatLimitReached(free) {
		t.Fatal("49 completed free frames should reach the limit")
	}
	free.OpponentScore = 23
	if snookerFormatLimitReached(free) {
		t.Fatal("48 completed free frames should remain playable")
	}
}

func TestRaceToTargetAutoCompletesFlexibleMatch(t *testing.T) {
	svcCtx := newSnookerActionTestSvc(t)
	match := seedSnookerActionMatch(t, svcCtx, 1060, 0, 303)
	match.SnookerFormat = model.SnookerFormatRaceTo
	match.SnookerTargetWins = 1
	if err := svcCtx.DB.Save(match).Error; err != nil {
		t.Fatalf("seed race-to match: %v", err)
	}
	resp, err := NewSnookerFrameActionLogic(matchLogicCtx(303), svcCtx).SnookerFrameAction(&types.SnookerFrameActionReq{
		MatchId: match.Id, Actor: 1, Action: model.SnookerFrameActionAwardFrame, Winner: 1,
		Reason: "判局", ClientActionId: "race-to-one", BaseRevision: 0,
	})
	if err != nil || !resp.Success || resp.Snapshot.Status != 2 || resp.Snapshot.MyScore != 1 {
		t.Fatalf("race-to target should auto complete: resp=%+v err=%v", resp, err)
	}
}

func TestFreeFormatFortyNinthFrameAutoCompletes(t *testing.T) {
	svcCtx := newSnookerActionTestSvc(t)
	match := seedSnookerActionMatch(t, svcCtx, 1061, 0, 303)
	match.SnookerFormat = model.SnookerFormatFree
	match.MyScore = 24
	match.OpponentScore = 24
	if err := svcCtx.DB.Save(match).Error; err != nil {
		t.Fatalf("seed free match: %v", err)
	}
	for roundNo := 1; roundNo <= 48; roundNo++ {
		winner := 1
		if roundNo%2 == 0 {
			winner = 2
		}
		if err := svcCtx.DB.Create(&model.MatchRound{MatchId: match.Id, RoundNo: roundNo, Winner: &winner, WinType: "complete"}).Error; err != nil {
			t.Fatalf("seed round %d: %v", roundNo, err)
		}
	}
	resp, err := NewSnookerFrameActionLogic(matchLogicCtx(303), svcCtx).SnookerFrameAction(&types.SnookerFrameActionReq{
		MatchId: match.Id, Actor: 1, Action: model.SnookerFrameActionAwardFrame, Winner: 1,
		Reason: "第49局", ClientActionId: "free-frame-49", BaseRevision: 0,
	})
	if err != nil || !resp.Success || resp.Snapshot.Status != 2 || resp.Snapshot.TotalRounds != 49 || resp.Snapshot.MyScore != 25 || resp.Snapshot.OpponentScore != 24 {
		t.Fatalf("49th free frame should auto complete: resp=%+v err=%v", resp, err)
	}
}
