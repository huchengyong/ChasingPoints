package match

import (
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
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
			snookerState, err = loadSnookerStateForMatch(svcCtx, match, roundNo)
			if err != nil {
				return state, err
			}
			if !match.CurrentFrameStarted && match.SnookerRulesVersion != model.SnookerRulesVersionWPBSA {
				snookerState = model.SnookerRoundState{ClearedColors: make([]int, 0, 6)}
			}
		}
	}

	state.CompletedRoundCount = roundCount
	state.SnookerState = snookerState
	state.Snapshot = buildMatchSyncSnapshotForUser(userId, match, roundCount, snookerState)
	state.Snapshot.LastAction = buildMatchLastAction(svcCtx, userId, match)

	// 填充裁判资料
	if capabilities := resolveMatchViewerCapabilities(match, userId); capabilities.RefereeBound && capabilities.RefereeUserId > 0 {
		if referee, err := svcCtx.UserModel.FindById(capabilities.RefereeUserId); err == nil && referee != nil {
			state.Snapshot.RefereeName = referee.Nickname
			state.Snapshot.RefereeAvatar = referee.Avatar
		}
	}
	if match.RefereeJoinedAt != nil {
		state.Snapshot.RefereeJoinedAt = match.RefereeJoinedAt.Format("2006-01-02T15:04:05+08:00")
	}

	return state, nil
}

func stringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
