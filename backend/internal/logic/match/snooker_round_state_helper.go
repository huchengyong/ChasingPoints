package match

import (
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
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

func buildSnookerStateForMatchFromActions(match *model.Match, actions []model.MatchAction, roundNo int) (model.SnookerRoundState, error) {
	if match == nil || match.GameType != 1 || roundNo <= 0 {
		return model.SnookerRoundState{ClearedColors: make([]int, 0, 6)}, nil
	}
	if match.SnookerRulesVersion != model.SnookerRulesVersionWPBSA {
		return model.BuildSnookerRoundState(actions, roundNo), nil
	}
	starter := model.SnookerStartingActor(match.StartingActor, roundNo)
	return model.ReplaySnookerRoundV2(actions, roundNo, starter)
}

func loadSnookerStateForMatch(svcCtx *svc.ServiceContext, match *model.Match, roundNo int) (model.SnookerRoundState, error) {
	if match == nil || match.GameType != 1 || roundNo <= 0 {
		return model.SnookerRoundState{ClearedColors: make([]int, 0, 6)}, nil
	}
	if svcCtx == nil || svcCtx.MatchModel == nil {
		return model.SnookerRoundState{}, nil
	}
	actions, err := svcCtx.MatchModel.ListActiveActions(match.Id)
	if err != nil {
		return model.SnookerRoundState{}, err
	}
	return buildSnookerStateForMatchFromActions(match, actions, roundNo)
}
