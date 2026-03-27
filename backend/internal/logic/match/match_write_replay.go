package match

import (
	"errors"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	mysqlDriver "github.com/go-sql-driver/mysql"
)

type matchWriteReplayState struct {
	Match          *model.Match
	ExistingAction *model.MatchAction
	View           matchWriteState
}

func reloadMatchWriteReplayState(svcCtx *svc.ServiceContext, userId, matchId int64, clientActionID string) (matchWriteReplayState, error) {
	state := matchWriteReplayState{}

	match, err := svcCtx.MatchModel.FindById(matchId)
	if err != nil {
		return state, err
	}
	state.Match = match

	if clientActionID != "" {
		action, actionErr := svcCtx.MatchModel.FindActionByClientActionID(matchId, clientActionID)
		if actionErr != nil {
			return state, actionErr
		}
		state.ExistingAction = action
	}

	view, viewErr := loadMatchWriteState(svcCtx, userId, match)
	if viewErr != nil {
		return state, viewErr
	}
	state.View = view

	return state, nil
}

func isRetryableMatchWriteError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, errRevisionConflict) {
		return true
	}

	var mysqlErr *mysqlDriver.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
