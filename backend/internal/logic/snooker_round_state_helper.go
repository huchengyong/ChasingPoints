package logic

import (
	"billiard_master/internal/model"
	"billiard_master/internal/svc"
)

func loadSnookerRoundState(svcCtx *svc.ServiceContext, matchId int64, roundNo int) (model.SnookerRoundState, error) {
	if svcCtx == nil || matchId == 0 || roundNo <= 0 {
		return model.SnookerRoundState{ClearedColors: make([]int, 0, 6)}, nil
	}

	actions, err := svcCtx.MatchModel.ListActiveActions(matchId)
	if err != nil {
		return model.SnookerRoundState{}, err
	}

	return model.BuildSnookerRoundState(actions, roundNo), nil
}
