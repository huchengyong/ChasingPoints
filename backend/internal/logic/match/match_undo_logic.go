package match

import (
	"context"
	"fmt"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg/ws"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type MatchUndoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 撤销操作
func NewMatchUndoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MatchUndoLogic {
	return &MatchUndoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MatchUndoLogic) MatchUndo(req *types.MatchUndoReq) (resp *types.MatchUndoResp, err error) {
	// 获取用户ID
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.MatchUndoResp{Success: false, Message: "用户未登录"}, nil
	}

	// 查询对局
	match, err := l.svcCtx.MatchModel.FindById(req.MatchId)
	if err != nil || match == nil {
		return &types.MatchUndoResp{Success: false, Message: "对局不存在"}, nil
	}

	// 验证用户权限和状态
	if match.Status != 1 {
		return &types.MatchUndoResp{Success: false, Message: "无法撤销"}, nil
	}
	if _, authorityErr := validateMatchWriteAuthority(match, userId); authorityErr != nil {
		if authorityErr == errMatchViewerNotParticipant {
			return &types.MatchUndoResp{Success: false, Message: "无法撤销"}, nil
		}
		view, stateErr := loadMatchWriteState(l.svcCtx, userId, match)
		if stateErr != nil {
			l.Logger.Errorf("加载权限拒绝快照失败: matchId=%d, userId=%d, err=%v", match.Id, userId, stateErr)
			return &types.MatchUndoResp{Success: false, Accepted: false, Message: authorityErr.Error()}, nil
		}
		scoreView := buildMatchWriteScoreView(userId, match)
		return &types.MatchUndoResp{
			Success:                   false,
			Accepted:                  false,
			Message:                   authorityErr.Error(),
			ServerRevision:            view.Snapshot.ServerRevision,
			Snapshot:                  view.Snapshot,
			CurrentFrameStarted:       match.CurrentFrameStarted,
			CurrentFrameMyScore:       scoreView.CurrentFrameMyScore,
			CurrentFrameOpponentScore: scoreView.CurrentFrameOpponentScore,
			MyScore:                   scoreView.MyScore,
			OpponentScore:             scoreView.OpponentScore,
		}, nil
	}
	if req.ClientActionId == "" {
		return &types.MatchUndoResp{Success: false, Accepted: false, Message: errMissingClientActionID.Error()}, nil
	}
	existingAction, err := l.svcCtx.MatchModel.FindActionByClientActionID(match.Id, req.ClientActionId)
	if err != nil {
		l.Logger.Errorf("查询幂等操作失败: matchId=%d, clientActionId=%s, err=%v", match.Id, req.ClientActionId, err)
		return &types.MatchUndoResp{Success: false, Accepted: false, Message: "撤销失败"}, nil
	}
	if existingAction != nil {
		view, stateErr := loadMatchWriteState(l.svcCtx, userId, match)
		if stateErr != nil {
			l.Logger.Errorf("加载重放快照失败: matchId=%d, clientActionId=%s, err=%v", match.Id, req.ClientActionId, stateErr)
			return &types.MatchUndoResp{Success: false, Accepted: false, Message: "撤销失败"}, nil
		}
		scoreView := buildMatchWriteScoreView(userId, match)
		return &types.MatchUndoResp{
			Accepted:                  true,
			Success:                   true,
			ClientActionId:            req.ClientActionId,
			ServerRevision:            view.Snapshot.ServerRevision,
			Snapshot:                  view.Snapshot,
			CurrentFrameStarted:       match.CurrentFrameStarted,
			CurrentFrameMyScore:       scoreView.CurrentFrameMyScore,
			CurrentFrameOpponentScore: scoreView.CurrentFrameOpponentScore,
			MyScore:                   scoreView.MyScore,
			OpponentScore:             scoreView.OpponentScore,
			Message:                   "已撤销上一步操作",
		}, nil
	}
	if err := validateMatchActionMeta(matchActionMeta{
		ClientActionID: req.ClientActionId,
		BaseRevision:   req.BaseRevision,
	}, match.SyncRevision); err != nil {
		view, stateErr := loadMatchWriteState(l.svcCtx, userId, match)
		if stateErr != nil {
			l.Logger.Errorf("加载冲突快照失败: matchId=%d, clientActionId=%s, err=%v", match.Id, req.ClientActionId, stateErr)
			return &types.MatchUndoResp{Success: false, Accepted: false, Message: err.Error()}, nil
		}
		scoreView := buildMatchWriteScoreView(userId, match)
		return &types.MatchUndoResp{
			Success:                   false,
			Accepted:                  false,
			Message:                   err.Error(),
			ClientActionId:            req.ClientActionId,
			ServerRevision:            view.Snapshot.ServerRevision,
			Snapshot:                  view.Snapshot,
			CurrentFrameStarted:       match.CurrentFrameStarted,
			CurrentFrameMyScore:       scoreView.CurrentFrameMyScore,
			CurrentFrameOpponentScore: scoreView.CurrentFrameOpponentScore,
			MyScore:                   scoreView.MyScore,
			OpponentScore:             scoreView.OpponentScore,
		}, nil
	}

	// 获取最后一条操作
	lastAction, err := l.svcCtx.MatchModel.GetLastAction(match.Id)
	if err != nil || lastAction == nil {
		return &types.MatchUndoResp{Success: false, Message: "没有可撤销的操作"}, nil
	}

	undoExtraData := fmt.Sprintf(`{"undone_action_id":%d}`, lastAction.Id)
	undoAction := &model.MatchAction{
		MatchId:        match.Id,
		RoundNo:        lastAction.RoundNo,
		ActionType:     "undo",
		Actor:          0,
		ScoreChange:    0,
		ClientActionId: stringPointer(req.ClientActionId),
		BaseRevision:   req.BaseRevision,
		ExtraData:      &undoExtraData,
		IsUndone:       1,
	}
	var serverRevision int64
	if isSnookerV2Match(match) {
		err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
			if err := undoSnookerV2ActionWithTx(l.svcCtx, tx, match, lastAction); err != nil {
				return err
			}
			revision, revisionErr := l.svcCtx.MatchModel.BumpMatchRevisionWithTx(tx, match)
			if revisionErr != nil {
				return revisionErr
			}
			serverRevision = revision
			return l.svcCtx.MatchModel.CreateActionWithRevisionWithTx(tx, undoAction, serverRevision)
		})
	} else {
		err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
			switch lastAction.ActionType {
			case "score":
				if match.GameType == 1 {
					if lastAction.Actor == 1 {
						match.CurrentFrameMyScore -= lastAction.ScoreChange
					} else {
						match.CurrentFrameOpponentScore -= lastAction.ScoreChange
					}
				} else if lastAction.Actor == 1 {
					match.MyScore -= lastAction.ScoreChange
				} else {
					match.OpponentScore -= lastAction.ScoreChange
				}
			case "foul":
				if match.GameType == 1 {
					if lastAction.Actor == 1 {
						match.CurrentFrameOpponentScore -= lastAction.ScoreChange
					} else {
						match.CurrentFrameMyScore -= lastAction.ScoreChange
					}
				} else if lastAction.Actor == 1 {
					match.OpponentScore -= lastAction.ScoreChange
				} else {
					match.MyScore -= lastAction.ScoreChange
				}
			case "win":
				if match.GameType == 1 {
					round, roundErr := l.svcCtx.MatchModel.GetLastRoundWithTx(tx, match.Id)
					if roundErr != nil {
						return roundErr
					}
					if round == nil || round.Winner == nil {
						return gorm.ErrRecordNotFound
					}
					reopenSnookerFrame(match, round, *round.Winner)
					if err := l.svcCtx.MatchModel.DeleteRoundWithTx(tx, round.Id); err != nil {
						return err
					}
				} else {
					if lastAction.Actor == 1 {
						match.MyScore -= lastAction.ScoreChange
					} else {
						match.OpponentScore -= lastAction.ScoreChange
					}
					round, roundErr := l.svcCtx.MatchModel.GetLastRoundWithTx(tx, match.Id)
					if roundErr != nil {
						return roundErr
					}
					if round != nil {
						if err := l.svcCtx.MatchModel.DeleteRoundWithTx(tx, round.Id); err != nil {
							return err
						}
					}
				}
			case "round_start":
				if match.GameType == 1 {
					match.CurrentFrameStarted = false
					match.CurrentFrameMyScore = 0
					match.CurrentFrameOpponentScore = 0
				}
			}

			if match.MyScore < 0 {
				match.MyScore = 0
			}
			if match.OpponentScore < 0 {
				match.OpponentScore = 0
			}
			if match.CurrentFrameMyScore < 0 {
				match.CurrentFrameMyScore = 0
			}
			if match.CurrentFrameOpponentScore < 0 {
				match.CurrentFrameOpponentScore = 0
			}

			if err := l.svcCtx.MatchModel.UndoActionWithTx(tx, lastAction.Id); err != nil {
				return err
			}
			revision, revisionErr := l.svcCtx.MatchModel.BumpMatchRevisionWithTx(tx, match)
			if revisionErr != nil {
				return revisionErr
			}
			serverRevision = revision
			return l.svcCtx.MatchModel.CreateActionWithRevisionWithTx(tx, undoAction, serverRevision)
		})
	}
	if err != nil {
		if isRetryableMatchWriteError(err) {
			replayState, replayErr := reloadMatchWriteReplayState(l.svcCtx, userId, req.MatchId, req.ClientActionId)
			if replayErr != nil {
				l.Logger.Errorf("重载写入快照失败: matchId=%d, clientActionId=%s, err=%v", req.MatchId, req.ClientActionId, replayErr)
				return &types.MatchUndoResp{Success: false, Accepted: false, Message: "撤销失败"}, nil
			}
			scoreView := buildMatchWriteScoreView(userId, replayState.Match)
			if replayState.ExistingAction != nil {
				return &types.MatchUndoResp{
					Accepted:                  true,
					Success:                   true,
					ClientActionId:            req.ClientActionId,
					ServerRevision:            replayState.View.Snapshot.ServerRevision,
					Snapshot:                  replayState.View.Snapshot,
					CurrentFrameStarted:       replayState.Match.CurrentFrameStarted,
					CurrentFrameMyScore:       scoreView.CurrentFrameMyScore,
					CurrentFrameOpponentScore: scoreView.CurrentFrameOpponentScore,
					MyScore:                   scoreView.MyScore,
					OpponentScore:             scoreView.OpponentScore,
					Message:                   "已撤销上一步操作",
				}, nil
			}
			return &types.MatchUndoResp{
				Accepted:                  false,
				Success:                   false,
				ClientActionId:            req.ClientActionId,
				ServerRevision:            replayState.View.Snapshot.ServerRevision,
				Snapshot:                  replayState.View.Snapshot,
				CurrentFrameStarted:       replayState.Match.CurrentFrameStarted,
				CurrentFrameMyScore:       scoreView.CurrentFrameMyScore,
				CurrentFrameOpponentScore: scoreView.CurrentFrameOpponentScore,
				MyScore:                   scoreView.MyScore,
				OpponentScore:             scoreView.OpponentScore,
				Message:                   errRevisionConflict.Error(),
			}, nil
		}
		l.Logger.Errorf("撤销失败: %v", err)
		return &types.MatchUndoResp{Success: false, Message: "撤销失败"}, nil
	}
	view, stateErr := loadMatchWriteState(l.svcCtx, userId, match)
	if stateErr != nil {
		l.Logger.Errorf("加载写入快照失败: matchId=%d, clientActionId=%s, err=%v", match.Id, req.ClientActionId, stateErr)
		return &types.MatchUndoResp{Success: false, Accepted: false, Message: "撤销失败"}, nil
	}

	// 获取当前局数
	roundCount, _ := l.svcCtx.MatchModel.GetRoundCount(match.Id)
	redBallCount := 0
	snookerState := view.SnookerState
	if match.GameType == 1 {
		redBallCount = snookerState.RedBallCount
	}

	// 推送 WebSocket 消息通知双方
	if isSnookerV2Match(match) {
		broadcastSnookerAction(match, snookerState, "undo", false)
	} else if ws.GlobalHub != nil {
		l.Logger.Infof("广播撤销结果: matchId=%d, action=%s, player1=%d, player2=%d, currentRound=%d",
			match.Id, lastAction.ActionType, match.MyScore, match.OpponentScore, int(roundCount)+1)
		ws.GlobalHub.BroadcastToMatch(match.Id, &ws.Message{
			Type: "score_update",
			Data: ws.ScoreUpdateData{
				MatchId:                       match.Id,
				ServerRevision:                view.Snapshot.ServerRevision,
				MyScore:                       match.MyScore,
				OpponentScore:                 match.OpponentScore,
				Player1Score:                  match.MyScore,
				Player2Score:                  match.OpponentScore,
				CurrentFramePlayer1Score:      match.CurrentFrameMyScore,
				CurrentFramePlayer2Score:      match.CurrentFrameOpponentScore,
				CurrentFrameStarted:           match.CurrentFrameStarted,
				CurrentRound:                  int(roundCount) + 1,
				TotalRounds:                   int(roundCount),
				ActionType:                    "undo",
				Actor:                         "",
				RedBallCount:                  redBallCount,
				SnookerClearanceStarted:       snookerState.ClearanceStarted,
				SnookerClearedColors:          snookerState.ClearedColors,
				SnookerExpectedClearanceScore: snookerState.ExpectedClearanceScore,
				SnookerClearanceCompleted:     snookerState.ClearanceCompleted,
				Status:                        match.Status,
			},
		})
	}

	l.Logger.Infof("用户 %d 撤销对局 %d 的操作: %s", userId, match.Id, lastAction.ActionType)

	scoreView := buildMatchWriteScoreView(userId, match)
	return &types.MatchUndoResp{
		Accepted:                  true,
		Success:                   true,
		ClientActionId:            req.ClientActionId,
		ServerRevision:            view.Snapshot.ServerRevision,
		Snapshot:                  view.Snapshot,
		CurrentFrameStarted:       match.CurrentFrameStarted,
		CurrentFrameMyScore:       scoreView.CurrentFrameMyScore,
		CurrentFrameOpponentScore: scoreView.CurrentFrameOpponentScore,
		MyScore:                   scoreView.MyScore,
		OpponentScore:             scoreView.OpponentScore,
		Message:                   "已撤销上一步操作",
	}, nil
}
