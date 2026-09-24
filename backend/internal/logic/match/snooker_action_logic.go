package match

import (
	"context"
	"errors"
	"fmt"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg/ws"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/gorm"
)

type snookerActionWriteInput struct {
	MatchID        int64
	ClientActionID string
	BaseRevision   int64
	Event          model.SnookerEvent
	ActionType     string
}

func isSnookerV2Match(match *model.Match) bool {
	return match != nil && match.GameType == 1 && match.SnookerRulesVersion == model.SnookerRulesVersionWPBSA
}

type snookerActionWriteResult struct {
	Match       *model.Match
	State       model.SnookerRoundState
	RoundEnded  bool
	FinishedNow bool
	Replayed    bool
	FinishReq   *types.FinishMatchReq
	Result      int
}

func executeSnookerAction(ctx context.Context, svcCtx *svc.ServiceContext, userID int64, input snookerActionWriteInput) (*types.SnookerActionResp, error) {
	if svcCtx == nil || svcCtx.DB == nil || svcCtx.MatchModel == nil || input.MatchID <= 0 {
		return &types.SnookerActionResp{Success: false, Message: "记录失败"}, nil
	}
	if input.ClientActionID == "" {
		return &types.SnookerActionResp{Success: false, Accepted: false, Message: errMissingClientActionID.Error()}, nil
	}

	match, err := svcCtx.MatchModel.FindById(input.MatchID)
	if err != nil || match == nil {
		return &types.SnookerActionResp{Success: false, Message: "对局不存在"}, nil
	}
	if _, authorityErr := validateMatchWriteAuthority(match, userID); authorityErr != nil {
		if authorityErr == errMatchViewerNotParticipant {
			return &types.SnookerActionResp{Success: false, Message: "无权操作本场对局"}, nil
		}
		return buildSnookerActionResponse(svcCtx, userID, match, input.ClientActionID, false, false, authorityErr.Error()), nil
	}
	if existing, findErr := svcCtx.MatchModel.FindActionByClientActionID(match.Id, input.ClientActionID); findErr != nil {
		return &types.SnookerActionResp{Success: false, Message: "查询操作状态失败"}, nil
	} else if existing != nil {
		same, compareErr := sameSnookerActionRequest(existing, input)
		if compareErr != nil {
			return buildSnookerActionResponse(svcCtx, userID, match, input.ClientActionID, false, false, "操作日志无法校验"), nil
		}
		if !same {
			return buildSnookerActionResponse(svcCtx, userID, match, input.ClientActionID, false, false, "操作编号已被其他动作使用"), nil
		}
		reconcileCompletedSnookerAction(ctx, svcCtx, userID, match, input)
		return buildSnookerActionResponse(svcCtx, userID, match, input.ClientActionID, true, true, ""), nil
	}
	if match.Status != 1 {
		return buildSnookerActionResponse(svcCtx, userID, match, input.ClientActionID, false, false, "对局已结束"), nil
	}
	if match.GameType != 1 || match.SnookerRulesVersion != model.SnookerRulesVersionWPBSA {
		return buildSnookerActionResponse(svcCtx, userID, match, input.ClientActionID, false, false, "当前对局不支持版本2斯诺克动作"), nil
	}
	if err := validateMatchActionMeta(matchActionMeta{ClientActionID: input.ClientActionID, BaseRevision: input.BaseRevision}, match.SyncRevision); err != nil {
		return buildSnookerActionResponse(svcCtx, userID, match, input.ClientActionID, false, false, err.Error()), nil
	}

	result := snookerActionWriteResult{}
	writeErr := svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		locked, err := svcCtx.MatchModel.FindByIdForUpdateWithTx(tx, input.MatchID)
		if err != nil {
			return err
		}
		if locked == nil {
			return gorm.ErrRecordNotFound
		}
		if existing, err := svcCtx.MatchModel.FindActionByClientActionIDWithTx(tx, locked.Id, input.ClientActionID); err != nil {
			return err
		} else if existing != nil {
			same, compareErr := sameSnookerActionRequest(existing, input)
			if compareErr != nil {
				return compareErr
			}
			if !same {
				return errors.New("操作编号已被其他动作使用")
			}
			result.Match = locked
			result.Replayed = true
			return nil
		}
		capabilities, authorityErr := validateMatchWriteAuthority(locked, userID)
		if authorityErr != nil {
			return authorityErr
		}
		if locked.Status != 1 {
			return errors.New("对局已结束")
		}
		if locked.GameType != 1 || locked.SnookerRulesVersion != model.SnookerRulesVersionWPBSA {
			return errors.New("当前对局不支持版本2斯诺克动作")
		}
		if !locked.CurrentFrameStarted {
			return errors.New("请先开始下一局")
		}
		if err := validateMatchActionMeta(matchActionMeta{ClientActionID: input.ClientActionID, BaseRevision: input.BaseRevision}, locked.SyncRevision); err != nil {
			return err
		}
		if input.Event.Kind == model.SnookerEventKindFrameAction && input.Event.FrameAction == model.SnookerFrameActionAwardFrame && capabilities.ViewerRole != matchViewerRoleReferee {
			return errors.New("只有裁判可以判给本局")
		}
		if input.Event.Kind == model.SnookerEventKindFrameAction &&
			(input.Event.FrameAction == model.SnookerFrameActionOfferConcession || input.Event.FrameAction == model.SnookerFrameActionAcceptConcession || input.Event.FrameAction == model.SnookerFrameActionRejectConcession) &&
			capabilities.ViewerRole != matchViewerRoleReferee && input.Event.Actor != resolveFinishActionActor(locked, userID) {
			return errors.New("只能提交本人认输决定")
		}

		roundCount, err := svcCtx.MatchModel.GetRoundCountWithTx(tx, locked.Id)
		if err != nil {
			return err
		}
		roundNo := int(roundCount) + 1
		actions, err := svcCtx.MatchModel.ListActiveActionsWithTx(tx, locked.Id)
		if err != nil {
			return err
		}
		starter := model.SnookerStartingActor(locked.StartingActor, roundNo)
		state, err := model.ReplaySnookerRoundV2(actions, roundNo, starter)
		if err != nil {
			return err
		}

		event := input.Event
		if event.Kind == model.SnookerEventKindStroke {
			event.VisitNo = state.VisitNo
		}
		if event.Kind == model.SnookerEventKindFrameAction &&
			(event.FrameAction == model.SnookerFrameActionAcceptConcession || event.FrameAction == model.SnookerFrameActionRejectConcession) {
			event.Scope = state.PendingConcessionScope
		}
		transition, err := model.ApplySnookerEvent(state, event)
		if err != nil {
			return err
		}
		extraData, err := model.EncodeSnookerEvent(event)
		if err != nil {
			return err
		}

		locked.CurrentFrameMyScore = transition.State.Player1Score
		locked.CurrentFrameOpponentScore = transition.State.Player2Score
		neededWins := 0
		if transition.State.FrameEnded {
			winner := transition.State.FrameWinner
			round := &model.MatchRound{
				MatchId:       locked.Id,
				RoundNo:       roundNo,
				MyScore:       transition.State.Player1Score,
				OpponentScore: transition.State.Player2Score,
				Winner:        &winner,
				WinType:       transition.State.FrameEndReason,
			}
			if err := svcCtx.MatchModel.CreateRoundWithTx(tx, round); err != nil {
				return err
			}
			if winner == 1 {
				locked.MyScore++
			} else {
				locked.OpponentScore++
			}
			locked.CurrentFrameStarted = false
			locked.CurrentFrameMyScore = 0
			locked.CurrentFrameOpponentScore = 0
			result.RoundEnded = true
			neededWins = locked.BestOfFrames/2 + 1
			matchConceded := transition.State.FrameEndReason == model.SnookerFrameEndMatchConcession ||
				(event.FrameAction == model.SnookerFrameActionAwardFrame && event.Scope == model.SnookerConcessionScopeMatch)
			if matchConceded {
				if transition.State.FrameWinner == 1 {
					locked.MyScore = neededWins
				} else {
					locked.OpponentScore = neededWins
				}
			}
		}

		revision, err := svcCtx.MatchModel.BumpMatchRevisionWithTx(tx, locked)
		if err != nil {
			return err
		}
		action := &model.MatchAction{
			MatchId:        locked.Id,
			RoundNo:        roundNo,
			ActionType:     input.ActionType,
			Actor:          event.Actor,
			ScoreChange:    transition.ScoreChange,
			ClientActionId: stringPointer(input.ClientActionID),
			BaseRevision:   input.BaseRevision,
			ExtraData:      &extraData,
		}
		if err := svcCtx.MatchModel.CreateActionWithRevisionWithTx(tx, action, revision); err != nil {
			return err
		}

		if transition.State.FrameEnded {
			if locked.MyScore >= neededWins || locked.OpponentScore >= neededWins {
				finishReq := &types.FinishMatchReq{
					MatchId:        locked.Id,
					ClientActionId: autoSnookerFinishActionID(input.ClientActionID),
					BaseRevision:   locked.SyncRevision,
				}
				finishLogic := NewFinishMatchLogic(ctx, svcCtx)
				settlement, err := finishLogic.settleMatchWithTx(tx, locked, userID, finishReq)
				if err != nil {
					return err
				}
				result.FinishedNow = true
				result.FinishReq = finishReq
				result.Result = settlement.Result
			}
		}
		result.Match = locked
		result.State = transition.State
		return nil
	})
	if writeErr != nil {
		fresh, _ := svcCtx.MatchModel.FindById(input.MatchID)
		message := writeErr.Error()
		if !errors.Is(writeErr, errRevisionConflict) && !errors.Is(writeErr, errMatchWriteForbiddenByReferee) && !errors.Is(writeErr, errMatchFinishPending) {
			if errors.Is(writeErr, gorm.ErrRecordNotFound) {
				message = "对局不存在"
			}
		}
		return buildSnookerActionResponse(svcCtx, userID, fresh, input.ClientActionID, false, false, message), nil
	}

	fresh, err := svcCtx.MatchModel.FindById(input.MatchID)
	if err != nil || fresh == nil {
		return &types.SnookerActionResp{Success: false, Message: "加载对局状态失败"}, nil
	}
	view, viewErr := loadMatchWriteState(svcCtx, userID, fresh)
	if viewErr != nil {
		return &types.SnookerActionResp{Success: false, Message: "加载对局快照失败"}, nil
	}
	if !result.Replayed {
		broadcastSnookerAction(fresh, view.SnookerState, input.ActionType, result.RoundEnded)
	}
	if result.Replayed {
		reconcileCompletedSnookerAction(ctx, svcCtx, userID, fresh, input)
	}
	if result.FinishedNow && result.FinishReq != nil {
		finishLogic := NewFinishMatchLogic(ctx, svcCtx)
		if _, err := finishLogic.finishMatchPostCommit(result.FinishReq, userID, fresh, result.Result); err != nil {
			finishLogic.Logger.Errorf("斯诺克自动结束后处理失败: matchId=%d err=%v", fresh.Id, err)
		}
	}
	return &types.SnookerActionResp{
		Accepted:       true,
		Success:        true,
		ClientActionId: input.ClientActionID,
		ServerRevision: view.Snapshot.ServerRevision,
		Snapshot:       view.Snapshot,
	}, nil
}

func sameSnookerActionRequest(existing *model.MatchAction, input snookerActionWriteInput) (bool, error) {
	if existing == nil || existing.ActionType != input.ActionType || existing.BaseRevision != input.BaseRevision {
		return false, nil
	}
	stored, err := model.DecodeSnookerEvent(existing.ExtraData)
	if err != nil {
		return false, err
	}
	expected := input.Event
	if expected.Kind == model.SnookerEventKindStroke {
		expected.VisitNo = stored.VisitNo
	}
	if expected.Kind == model.SnookerEventKindFrameAction &&
		(expected.FrameAction == model.SnookerFrameActionAcceptConcession || expected.FrameAction == model.SnookerFrameActionRejectConcession) {
		if expected.Scope != "" && expected.Scope != stored.Scope {
			return false, nil
		}
		expected.Scope = stored.Scope
	}
	raw, err := model.EncodeSnookerEvent(expected)
	if err != nil {
		return false, nil
	}
	return existing.ExtraData != nil && raw == *existing.ExtraData, nil
}

func reconcileCompletedSnookerAction(ctx context.Context, svcCtx *svc.ServiceContext, userID int64, match *model.Match, input snookerActionWriteInput) {
	if match == nil || match.Status != 2 || match.Result == nil {
		return
	}
	finishLogic := NewFinishMatchLogic(ctx, svcCtx)
	if _, err := finishLogic.finishMatchPostCommit(&types.FinishMatchReq{
		MatchId:        match.Id,
		ClientActionId: input.ClientActionID,
		BaseRevision:   input.BaseRevision,
	}, userID, match, *match.Result); err != nil {
		finishLogic.Logger.Errorf("斯诺克幂等重放补偿失败: matchId=%d err=%v", match.Id, err)
	}
}

func buildSnookerActionResponse(svcCtx *svc.ServiceContext, userID int64, match *model.Match, clientActionID string, accepted, success bool, message string) *types.SnookerActionResp {
	resp := &types.SnookerActionResp{
		Accepted:       accepted,
		Success:        success,
		ClientActionId: clientActionID,
		Message:        message,
	}
	if match == nil {
		return resp
	}
	if view, err := loadMatchWriteState(svcCtx, userID, match); err == nil {
		resp.ServerRevision = view.Snapshot.ServerRevision
		resp.Snapshot = view.Snapshot
	}
	return resp
}

func autoSnookerFinishActionID(clientActionID string) string {
	const suffix = ":end"
	maxPrefix := 64 - len(suffix)
	if len(clientActionID) > maxPrefix {
		clientActionID = clientActionID[:maxPrefix]
	}
	return clientActionID + suffix
}

func broadcastSnookerAction(match *model.Match, state model.SnookerRoundState, actionType string, roundEnded bool) {
	if ws.GlobalHub == nil || match == nil {
		return
	}
	messageType := "score_update"
	actor := state.Striker
	currentRound := state.RoundNo
	if roundEnded {
		messageType = "round_end"
		actor = state.FrameWinner
		if match.Status == 1 {
			currentRound++
		}
	}
	ws.GlobalHub.BroadcastToMatch(match.Id, &ws.Message{
		Type: messageType,
		Data: ws.ScoreUpdateData{
			MatchId:                       match.Id,
			ServerRevision:                match.SyncRevision,
			MyScore:                       match.MyScore,
			OpponentScore:                 match.OpponentScore,
			Player1Score:                  match.MyScore,
			Player2Score:                  match.OpponentScore,
			CurrentFramePlayer1Score:      match.CurrentFrameMyScore,
			CurrentFramePlayer2Score:      match.CurrentFrameOpponentScore,
			CurrentFrameStarted:           match.CurrentFrameStarted,
			CurrentRound:                  currentRound,
			TotalRounds:                   match.MyScore + match.OpponentScore,
			RoundNumber:                   state.RoundNo,
			RoundPlayer1Score:             state.Player1Score,
			RoundPlayer2Score:             state.Player2Score,
			Winner:                        state.FrameWinner,
			ActionType:                    actionType,
			Actor:                         fmt.Sprintf("player%d", actor),
			RedBallCount:                  state.RedBallCount,
			SnookerClearanceStarted:       state.ClearanceStarted,
			SnookerClearedColors:          state.ClearedColors,
			SnookerExpectedClearanceScore: state.ExpectedClearanceScore,
			SnookerClearanceCompleted:     state.ClearanceCompleted,
			SnookerRulesVersion:           match.SnookerRulesVersion,
			BestOfFrames:                  match.BestOfFrames,
			StartingActor:                 match.StartingActor,
			SnookerPhase:                  state.Phase,
			SnookerBallOn:                 state.BallOn,
			SnookerStriker:                state.Striker,
			SnookerVisitNo:                state.VisitNo,
			SnookerCurrentBreak:           state.CurrentBreak,
			SnookerRedsRemaining:          state.RedsRemaining,
			SnookerFreeBallAvailable:      state.FreeBallAvailable,
			SnookerCueBallInHand:          state.CueBallInHand,
			SnookerMissWarningActive:      state.MissWarningActive,
			SnookerRespottedBlackPending:  state.Phase == model.SnookerPhaseRespottedBlackPending,
			SnookerPendingConcessionActor: state.PendingConcessionActor,
			SnookerPendingConcessionScope: state.PendingConcessionScope,
			SnookerFrameEndReason:         state.FrameEndReason,
			Status:                        match.Status,
		},
	})
}
