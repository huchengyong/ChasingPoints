package logic

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg/ws"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type MatchFoulLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 犯规
func NewMatchFoulLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MatchFoulLogic {
	return &MatchFoulLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MatchFoulLogic) MatchFoul(req *types.MatchFoulReq) (resp *types.MatchScoreResp, err error) {
	// 获取用户ID
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.MatchScoreResp{Success: false}, nil
	}

	// 查询对局
	match, err := l.svcCtx.MatchModel.FindById(req.MatchId)
	if err != nil {
		l.Logger.Errorf("查询对局失败: matchId=%d, err=%v", req.MatchId, err)
		return &types.MatchScoreResp{Success: false}, nil
	}
	if match == nil {
		l.Logger.Errorf("对局不存在: matchId=%d", req.MatchId)
		return &types.MatchScoreResp{Success: false}, nil
	}

	// 验证用户权限和状态
	isPlayer1 := match.UserId == userId
	isPlayer2 := match.OpponentId != nil && *match.OpponentId == userId
	if !isPlayer1 && !isPlayer2 {
		l.Logger.Errorf("用户不是对局参与者: matchUserId=%d, matchOpponentId=%v, currentUserId=%d",
			match.UserId, match.OpponentId, userId)
		return &types.MatchScoreResp{Success: false}, nil
	}
	if match.Status != 1 {
		l.Logger.Errorf("对局状态不是进行中: matchId=%d, status=%d", match.Id, match.Status)
		return &types.MatchScoreResp{Success: false}, nil
	}
	if req.ClientActionId == "" {
		return &types.MatchScoreResp{
			Success:  false,
			Accepted: false,
			Message:  errMissingClientActionID.Error(),
		}, nil
	}
	existingAction, err := l.svcCtx.MatchModel.FindActionByClientActionID(match.Id, req.ClientActionId)
	if err != nil {
		l.Logger.Errorf("查询幂等操作失败: matchId=%d, clientActionId=%s, err=%v", match.Id, req.ClientActionId, err)
		return &types.MatchScoreResp{Success: false, Accepted: false}, nil
	}
	if existingAction != nil {
		view, stateErr := loadMatchWriteState(l.svcCtx, userId, match)
		if stateErr != nil {
			l.Logger.Errorf("加载重放快照失败: matchId=%d, clientActionId=%s, err=%v", match.Id, req.ClientActionId, stateErr)
			return &types.MatchScoreResp{Success: false, Accepted: false}, nil
		}
		scoreView := buildMatchWriteScoreView(userId, match)
		return &types.MatchScoreResp{
			Accepted:                      true,
			Success:                       true,
			ClientActionId:                req.ClientActionId,
			ServerRevision:                view.Snapshot.ServerRevision,
			Snapshot:                      view.Snapshot,
			RedBallCount:                  view.SnookerState.RedBallCount,
			SnookerClearanceStarted:       view.SnookerState.ClearanceStarted,
			SnookerClearedColors:          view.SnookerState.ClearedColors,
			SnookerExpectedClearanceScore: view.SnookerState.ExpectedClearanceScore,
			SnookerClearanceCompleted:     view.SnookerState.ClearanceCompleted,
			CurrentFrameStarted:           match.CurrentFrameStarted,
			CurrentFrameMyScore:           scoreView.CurrentFrameMyScore,
			CurrentFrameOpponentScore:     scoreView.CurrentFrameOpponentScore,
			MyScore:                       scoreView.MyScore,
			OpponentScore:                 scoreView.OpponentScore,
		}, nil
	}
	if err := validateMatchActionMeta(matchActionMeta{
		ClientActionID: req.ClientActionId,
		BaseRevision:   req.BaseRevision,
	}, match.SyncRevision); err != nil {
		view, stateErr := loadMatchWriteState(l.svcCtx, userId, match)
		if stateErr != nil {
			l.Logger.Errorf("加载冲突快照失败: matchId=%d, clientActionId=%s, err=%v", match.Id, req.ClientActionId, stateErr)
			return &types.MatchScoreResp{Success: false, Accepted: false, Message: err.Error()}, nil
		}
		scoreView := buildMatchWriteScoreView(userId, match)
		return &types.MatchScoreResp{
			Success:                       false,
			Accepted:                      false,
			Message:                       err.Error(),
			ClientActionId:                req.ClientActionId,
			ServerRevision:                view.Snapshot.ServerRevision,
			Snapshot:                      view.Snapshot,
			RedBallCount:                  view.SnookerState.RedBallCount,
			SnookerClearanceStarted:       view.SnookerState.ClearanceStarted,
			SnookerClearedColors:          view.SnookerState.ClearedColors,
			SnookerExpectedClearanceScore: view.SnookerState.ExpectedClearanceScore,
			SnookerClearanceCompleted:     view.SnookerState.ClearanceCompleted,
			CurrentFrameStarted:           match.CurrentFrameStarted,
			CurrentFrameMyScore:           scoreView.CurrentFrameMyScore,
			CurrentFrameOpponentScore:     scoreView.CurrentFrameOpponentScore,
			MyScore:                       scoreView.MyScore,
			OpponentScore:                 scoreView.OpponentScore,
		}, nil
	}

	score := req.Score
	if score <= 0 {
		score = 1
	}
	roundCount, _ := l.svcCtx.MatchModel.GetRoundCount(match.Id)
	currentRound := int(roundCount) + 1

	if match.GameType == 1 && !match.CurrentFrameStarted {
		return &types.MatchScoreResp{
			Success: false,
			Message: "请先开始下一局",
		}, nil
	}

	// 犯规时对方加对应分数
	if match.GameType == 1 {
		applySnookerFoul(match, req.Actor, score)
	} else {
		if req.Actor == 1 {
			match.OpponentScore += score
		} else {
			match.MyScore += score
		}
	}
	action := &model.MatchAction{
		MatchId:        match.Id,
		RoundNo:        currentRound,
		ActionType:     "foul",
		Actor:          req.Actor,
		ScoreChange:    score,
		ClientActionId: stringPointer(req.ClientActionId),
		BaseRevision:   req.BaseRevision,
	}
	var serverRevision int64
	if err := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		revision, revisionErr := l.svcCtx.MatchModel.BumpMatchRevisionWithTx(tx, match)
		if revisionErr != nil {
			return revisionErr
		}
		serverRevision = revision
		if err := l.svcCtx.MatchModel.CreateActionWithRevisionWithTx(tx, action, serverRevision); err != nil {
			return err
		}
		return nil
	}); err != nil {
		if isRetryableMatchWriteError(err) {
			replayState, replayErr := reloadMatchWriteReplayState(l.svcCtx, userId, req.MatchId, req.ClientActionId)
			if replayErr != nil {
				l.Logger.Errorf("重载写入快照失败: matchId=%d, clientActionId=%s, err=%v", req.MatchId, req.ClientActionId, replayErr)
				return &types.MatchScoreResp{Success: false, Accepted: false}, nil
			}
			scoreView := buildMatchWriteScoreView(userId, replayState.Match)
			if replayState.ExistingAction != nil {
				return &types.MatchScoreResp{
					Accepted:                      true,
					Success:                       true,
					ClientActionId:                req.ClientActionId,
					ServerRevision:                replayState.View.Snapshot.ServerRevision,
					Snapshot:                      replayState.View.Snapshot,
					RedBallCount:                  replayState.View.SnookerState.RedBallCount,
					SnookerClearanceStarted:       replayState.View.SnookerState.ClearanceStarted,
					SnookerClearedColors:          replayState.View.SnookerState.ClearedColors,
					SnookerExpectedClearanceScore: replayState.View.SnookerState.ExpectedClearanceScore,
					SnookerClearanceCompleted:     replayState.View.SnookerState.ClearanceCompleted,
					CurrentFrameStarted:           replayState.Match.CurrentFrameStarted,
					CurrentFrameMyScore:           scoreView.CurrentFrameMyScore,
					CurrentFrameOpponentScore:     scoreView.CurrentFrameOpponentScore,
					MyScore:                       scoreView.MyScore,
					OpponentScore:                 scoreView.OpponentScore,
				}, nil
			}
			return &types.MatchScoreResp{
				Accepted:                      false,
				Success:                       false,
				Message:                       errRevisionConflict.Error(),
				ClientActionId:                req.ClientActionId,
				ServerRevision:                replayState.View.Snapshot.ServerRevision,
				Snapshot:                      replayState.View.Snapshot,
				RedBallCount:                  replayState.View.SnookerState.RedBallCount,
				SnookerClearanceStarted:       replayState.View.SnookerState.ClearanceStarted,
				SnookerClearedColors:          replayState.View.SnookerState.ClearedColors,
				SnookerExpectedClearanceScore: replayState.View.SnookerState.ExpectedClearanceScore,
				SnookerClearanceCompleted:     replayState.View.SnookerState.ClearanceCompleted,
				CurrentFrameStarted:           replayState.Match.CurrentFrameStarted,
				CurrentFrameMyScore:           scoreView.CurrentFrameMyScore,
				CurrentFrameOpponentScore:     scoreView.CurrentFrameOpponentScore,
				MyScore:                       scoreView.MyScore,
				OpponentScore:                 scoreView.OpponentScore,
			}, nil
		}
		l.Logger.Errorf("更新分数失败: %v", err)
		return &types.MatchScoreResp{Success: false}, nil
	}

	// 先立即推送 WebSocket 消息（优化响应延迟）
	actor := "me"
	if req.Actor == 2 {
		actor = "opponent"
	}

	redBallCount := 0
	snookerState := model.SnookerRoundState{ClearedColors: make([]int, 0, 6)}
	if match.GameType == 1 {
		if state, stateErr := loadSnookerRoundState(l.svcCtx, match.Id, currentRound); stateErr == nil {
			snookerState = state
			redBallCount = state.RedBallCount
		}
	}
	view, stateErr := loadMatchWriteState(l.svcCtx, userId, match)
	if stateErr != nil {
		l.Logger.Errorf("加载写入快照失败: matchId=%d, clientActionId=%s, err=%v", match.Id, req.ClientActionId, stateErr)
		return &types.MatchScoreResp{Success: false, Accepted: false}, nil
	}

	if ws.GlobalHub != nil {
		l.Logger.Infof("广播比分更新: matchId=%d, action=foul, actor=%s, player1=%d, player2=%d, currentRound=%d",
			match.Id, actor, match.MyScore, match.OpponentScore, currentRound)
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
				CurrentRound:                  currentRound,
				ActionType:                    "foul",
				Actor:                         actor,
				RedBallCount:                  redBallCount,
				SnookerClearanceStarted:       snookerState.ClearanceStarted,
				SnookerClearedColors:          snookerState.ClearedColors,
				SnookerExpectedClearanceScore: snookerState.ExpectedClearanceScore,
				SnookerClearanceCompleted:     snookerState.ClearanceCompleted,
				Status:                        match.Status,
			},
		})
	}

	scoreView := buildMatchWriteScoreView(userId, match)
	return &types.MatchScoreResp{
		Accepted:                      true,
		Success:                       true,
		ClientActionId:                req.ClientActionId,
		ServerRevision:                view.Snapshot.ServerRevision,
		Snapshot:                      view.Snapshot,
		RedBallCount:                  view.SnookerState.RedBallCount,
		SnookerClearanceStarted:       view.SnookerState.ClearanceStarted,
		SnookerClearedColors:          view.SnookerState.ClearedColors,
		SnookerExpectedClearanceScore: view.SnookerState.ExpectedClearanceScore,
		SnookerClearanceCompleted:     view.SnookerState.ClearanceCompleted,
		CurrentFrameStarted:           match.CurrentFrameStarted,
		CurrentFrameMyScore:           scoreView.CurrentFrameMyScore,
		CurrentFrameOpponentScore:     scoreView.CurrentFrameOpponentScore,
		MyScore:                       scoreView.MyScore,
		OpponentScore:                 scoreView.OpponentScore,
	}, nil
}
