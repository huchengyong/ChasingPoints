package logic

import (
	"billiard_master/internal/model"
	"billiard_master/internal/svc"
	"billiard_master/internal/types"
)

type matchWriteState struct {
	CompletedRoundCount int64
	SnookerState        model.SnookerRoundState
	Snapshot            types.MatchSyncSnapshot
}

func loadMatchWriteState(svcCtx *svc.ServiceContext, userId int64, match *model.Match) (matchWriteState, error) {
	state := matchWriteState{}
	if match == nil {
		return state, nil
	}

	roundCount, err := svcCtx.MatchModel.GetRoundCount(match.Id)
	if err != nil {
		return state, err
	}

	snookerState := model.SnookerRoundState{ClearedColors: make([]int, 0, 6)}
	if match.GameType == 1 {
		roundNo := int(roundCount) + 1
		if !match.CurrentFrameStarted && match.Status == 1 {
			roundNo = int(roundCount)
		}
		if roundNo > 0 {
			actions, actionsErr := svcCtx.MatchModel.ListActiveActions(match.Id)
			if actionsErr != nil {
				return state, actionsErr
			}
			snookerState = model.BuildSnookerRoundState(actions, roundNo)
			if !match.CurrentFrameStarted {
				snookerState = model.SnookerRoundState{ClearedColors: make([]int, 0, 6)}
			}
		}
	}

	state.CompletedRoundCount = roundCount
	state.SnookerState = snookerState
	state.Snapshot = buildMatchSyncSnapshotForUser(userId, match, roundCount, snookerState)
	return state, nil
}

func stringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
