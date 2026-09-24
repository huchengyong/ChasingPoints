package match

import (
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func buildMatchSyncSnapshotForUser(userId int64, match *model.Match, completedRoundCount int64, snookerState model.SnookerRoundState) types.MatchSyncSnapshot {
	if match == nil {
		return types.MatchSyncSnapshot{}
	}

	capabilities := resolveMatchViewerCapabilities(match, userId)
	myScore := match.MyScore
	opponentScore := match.OpponentScore
	currentFrameMyScore := match.CurrentFrameMyScore
	currentFrameOpponentScore := match.CurrentFrameOpponentScore

	if shouldUsePlayer2Perspective(capabilities.ViewerRole) {
		myScore = match.OpponentScore
		opponentScore = match.MyScore
		currentFrameMyScore = match.CurrentFrameOpponentScore
		currentFrameOpponentScore = match.CurrentFrameMyScore
	}

	currentRound := int(completedRoundCount) + 1
	if match.Status != 1 && !match.CurrentFrameStarted {
		currentRound = int(completedRoundCount)
	}

	refereeDurationSeconds := int64(0)
	if match.RefereeJoinedAt != nil {
		if match.EndTime != nil {
			refereeDurationSeconds = int64(match.EndTime.Sub(*match.RefereeJoinedAt).Seconds())
		} else {
			refereeDurationSeconds = int64(time.Since(*match.RefereeJoinedAt).Seconds())
		}
		if refereeDurationSeconds < 0 {
			refereeDurationSeconds = 0
		}
	}

	canFinish := capabilities.CanFinish
	canRequestFinish := capabilities.CanRequestFinish
	canConfirmFinish := capabilities.CanConfirmFinish
	canDisputeFinish := capabilities.CanDisputeFinish
	canWithdrawFinish := capabilities.CanWithdrawFinish
	if match.GameType == 1 && match.SnookerRulesVersion == model.SnookerRulesVersionWPBSA {
		canFinish = false
		canRequestFinish = false
		canConfirmFinish = false
		canDisputeFinish = false
		canWithdrawFinish = false
	}

	return types.MatchSyncSnapshot{
		MatchId:                       match.Id,
		Status:                        match.Status,
		ServerRevision:                match.SyncRevision,
		MatchMode:                     model.NormalizeMatchMode(match.MatchMode),
		Visibility:                    model.NormalizeMatchVisibility(match.Visibility, match.MatchMode),
		FinishState:                   model.NormalizeFinishState(match.FinishState),
		FinishRequestedBy:             resolveFinishRequestedBy(match),
		ViewerRole:                    capabilities.ViewerRole,
		RefereeBound:                  capabilities.RefereeBound,
		RefereeUserId:                 capabilities.RefereeUserId,
		CompletedByUserId:             resolveCompletedByUserId(match),
		CompletionSource:              resolveCompletionSource(match),
		RefereeDurationSeconds:        refereeDurationSeconds,
		CanScore:                      capabilities.CanScore,
		CanUndo:                       capabilities.CanUndo,
		CanFinish:                     canFinish,
		CanRequestFinish:              canRequestFinish,
		CanConfirmFinish:              canConfirmFinish,
		CanDisputeFinish:              canDisputeFinish,
		CanWithdrawFinish:             canWithdrawFinish,
		MyScore:                       myScore,
		OpponentScore:                 opponentScore,
		CurrentFrameStarted:           match.CurrentFrameStarted,
		CurrentFrameMyScore:           currentFrameMyScore,
		CurrentFrameOpponentScore:     currentFrameOpponentScore,
		CurrentRound:                  currentRound,
		TotalRounds:                   int(completedRoundCount),
		RedBallCount:                  snookerState.RedBallCount,
		SnookerClearanceStarted:       snookerState.ClearanceStarted,
		SnookerClearedColors:          snookerState.ClearedColors,
		SnookerExpectedClearanceScore: snookerState.ExpectedClearanceScore,
		SnookerClearanceCompleted:     snookerState.ClearanceCompleted,
		SnookerRulesVersion:           match.SnookerRulesVersion,
		BestOfFrames:                  match.BestOfFrames,
		StartingActor:                 match.StartingActor,
		SnookerPhase:                  snookerState.Phase,
		SnookerBallOn:                 snookerState.BallOn,
		SnookerStriker:                snookerState.Striker,
		SnookerVisitNo:                snookerState.VisitNo,
		SnookerCurrentBreak:           snookerState.CurrentBreak,
		SnookerRedsRemaining:          snookerState.RedsRemaining,
		SnookerFreeBallAvailable:      snookerState.FreeBallAvailable,
		SnookerCueBallInHand:          snookerState.CueBallInHand,
		SnookerMissWarningActive:      snookerState.MissWarningActive,
		SnookerRespottedBlackPending:  snookerState.Phase == model.SnookerPhaseRespottedBlackPending,
		SnookerPendingConcessionActor: snookerState.PendingConcessionActor,
		SnookerPendingConcessionScope: snookerState.PendingConcessionScope,
		SnookerFrameEndReason:         snookerState.FrameEndReason,
	}
}
