package match

import (
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

	return types.MatchSyncSnapshot{
		MatchId:                       match.Id,
		Status:                        match.Status,
		ServerRevision:                match.SyncRevision,
		ViewerRole:                    capabilities.ViewerRole,
		RefereeBound:                  capabilities.RefereeBound,
		RefereeUserId:                 capabilities.RefereeUserId,
		CanScore:                      capabilities.CanScore,
		CanUndo:                       capabilities.CanUndo,
		CanFinish:                     capabilities.CanFinish,
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
	}
}
