package match

import (
	"errors"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/gorm"
)

func undoSnookerV2ActionWithTx(svcCtx *svc.ServiceContext, tx *gorm.DB, match *model.Match, lastAction *model.MatchAction) error {
	if svcCtx == nil || svcCtx.MatchModel == nil || match == nil || lastAction == nil {
		return errors.New("撤销参数不完整")
	}
	switch lastAction.ActionType {
	case "round_start":
		if err := svcCtx.MatchModel.UndoActionWithTx(tx, lastAction.Id); err != nil {
			return err
		}
		match.CurrentFrameStarted = false
		match.CurrentFrameMyScore = 0
		match.CurrentFrameOpponentScore = 0
		return nil
	case model.MatchActionTypeSnookerStroke, model.MatchActionTypeSnookerFrameAction:
	default:
		return errors.New("当前操作不可撤销")
	}

	if err := svcCtx.MatchModel.UndoActionWithTx(tx, lastAction.Id); err != nil {
		return err
	}
	if !match.CurrentFrameStarted {
		round, err := svcCtx.MatchModel.GetLastRoundWithTx(tx, match.Id)
		if err != nil {
			return err
		}
		if round == nil || round.RoundNo != lastAction.RoundNo || round.Winner == nil {
			return errors.New("找不到需要回退的局记录")
		}
		if *round.Winner == 1 {
			match.MyScore--
		} else {
			match.OpponentScore--
		}
		if err := svcCtx.MatchModel.DeleteRoundWithTx(tx, round.Id); err != nil {
			return err
		}
		match.CurrentFrameStarted = true
	}

	actions, err := svcCtx.MatchModel.ListActiveActionsWithTx(tx, match.Id)
	if err != nil {
		return err
	}
	starter := model.SnookerStartingActor(match.StartingActor, lastAction.RoundNo)
	state, err := model.ReplaySnookerRoundV2(actions, lastAction.RoundNo, starter)
	if err != nil {
		return err
	}
	if state.FrameEnded {
		return errors.New("撤销后当前局仍处于结束状态")
	}
	match.CurrentFrameStarted = true
	match.CurrentFrameMyScore = state.Player1Score
	match.CurrentFrameOpponentScore = state.Player2Score
	if match.MyScore < 0 {
		match.MyScore = 0
	}
	if match.OpponentScore < 0 {
		match.OpponentScore = 0
	}
	return nil
}
