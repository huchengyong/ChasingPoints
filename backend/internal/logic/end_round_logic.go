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

type EndRoundLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 结束一局
func NewEndRoundLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EndRoundLogic {
	return &EndRoundLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EndRoundLogic) EndRound(req *types.EndRoundReq) (resp *types.EndRoundResp, err error) {
	// 获取用户ID
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.EndRoundResp{Success: false}, nil
	}

	// 查询对局
	match, err := l.svcCtx.MatchModel.FindById(req.MatchId)
	if err != nil || match == nil {
		l.Logger.Errorf("对局不存在: %v", err)
		return &types.EndRoundResp{Success: false}, nil
	}

	// 验证用户权限
	isPlayer1 := match.UserId == userId
	isPlayer2 := match.OpponentId != nil && *match.OpponentId == userId
	if !isPlayer1 && !isPlayer2 {
		l.Logger.Errorf("用户不是对局参与者: matchUserId=%d, matchOpponentId=%v, currentUserId=%d",
			match.UserId, match.OpponentId, userId)
		return &types.EndRoundResp{Success: false}, nil
	}
	if req.ClientActionId == "" {
		return &types.EndRoundResp{
			Success:  false,
			Accepted: false,
		}, nil
	}
	existingAction, err := l.svcCtx.MatchModel.FindActionByClientActionID(match.Id, req.ClientActionId)
	if err != nil {
		l.Logger.Errorf("查询幂等操作失败: matchId=%d, clientActionId=%s, err=%v", match.Id, req.ClientActionId, err)
		return &types.EndRoundResp{Success: false, Accepted: false}, nil
	}
	if existingAction != nil {
		view, stateErr := loadMatchWriteState(l.svcCtx, userId, match)
		if stateErr != nil {
			l.Logger.Errorf("加载重放快照失败: matchId=%d, clientActionId=%s, err=%v", match.Id, req.ClientActionId, stateErr)
			return &types.EndRoundResp{Success: false, Accepted: false}, nil
		}
		scoreView := buildMatchWriteScoreView(userId, match)
		return &types.EndRoundResp{
			Accepted:                  true,
			Success:                   true,
			ClientActionId:            req.ClientActionId,
			ServerRevision:            view.Snapshot.ServerRevision,
			Snapshot:                  view.Snapshot,
			RoundNo:                   existingAction.RoundNo,
			CurrentFrameStarted:       match.CurrentFrameStarted,
			CurrentFrameMyScore:       scoreView.CurrentFrameMyScore,
			CurrentFrameOpponentScore: scoreView.CurrentFrameOpponentScore,
			MyScore:                   scoreView.MyScore,
			OpponentScore:             scoreView.OpponentScore,
		}, nil
	}
	if err := validateMatchActionMeta(matchActionMeta{
		ClientActionID: req.ClientActionId,
		BaseRevision:   req.BaseRevision,
	}, match.SyncRevision); err != nil {
		view, stateErr := loadMatchWriteState(l.svcCtx, userId, match)
		if stateErr != nil {
			l.Logger.Errorf("加载冲突快照失败: matchId=%d, clientActionId=%s, err=%v", match.Id, req.ClientActionId, stateErr)
			return &types.EndRoundResp{Success: false, Accepted: false}, nil
		}
		scoreView := buildMatchWriteScoreView(userId, match)
		return &types.EndRoundResp{
			Success:                   false,
			Accepted:                  false,
			ClientActionId:            req.ClientActionId,
			ServerRevision:            view.Snapshot.ServerRevision,
			Snapshot:                  view.Snapshot,
			RoundNo:                   int(view.CompletedRoundCount),
			CurrentFrameStarted:       match.CurrentFrameStarted,
			CurrentFrameMyScore:       scoreView.CurrentFrameMyScore,
			CurrentFrameOpponentScore: scoreView.CurrentFrameOpponentScore,
			MyScore:                   scoreView.MyScore,
			OpponentScore:             scoreView.OpponentScore,
		}, nil
	}

	// 验证对局状态
	if match.Status != 1 {
		return &types.EndRoundResp{Success: false}, nil
	}

	// 获取当前局数
	roundCount, _ := l.svcCtx.MatchModel.GetRoundCount(match.Id)
	newRoundNo := int(roundCount) + 1

	if match.GameType == 1 && !match.CurrentFrameStarted {
		return &types.EndRoundResp{Success: false}, nil
	}

	round := &model.MatchRound{
		MatchId: match.Id,
		RoundNo: newRoundNo,
		Winner:  &req.Winner,
		WinType: req.WinType,
	}
	scoreChange := req.Score
	if match.GameType == 1 {
		finishedRound := finalizeSnookerFrame(match, req.Winner)
		round.MyScore = finishedRound.MyScore
		round.OpponentScore = finishedRound.OpponentScore
		scoreChange = 1
	} else {
		round.MyScore = match.MyScore
		round.OpponentScore = match.OpponentScore
		if req.Score > 0 {
			if req.Winner == 1 {
				match.MyScore += req.Score
			} else {
				match.OpponentScore += req.Score
			}
		}
	}

	extraData := `{"win_type":"` + req.WinType + `"}`
	action := &model.MatchAction{
		MatchId:        match.Id,
		RoundNo:        newRoundNo,
		ActionType:     "win",
		Actor:          req.Winner,
		ScoreChange:    scoreChange,
		ClientActionId: stringPointer(req.ClientActionId),
		BaseRevision:   req.BaseRevision,
		ExtraData:      &extraData,
	}
	var serverRevision int64

	if err := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		revision, revisionErr := l.svcCtx.MatchModel.BumpMatchRevisionWithTx(tx, match)
		if revisionErr != nil {
			return revisionErr
		}
		serverRevision = revision
		if err := l.svcCtx.MatchModel.CreateRoundWithTx(tx, round); err != nil {
			return err
		}
		if achievementType := normalizeStoredAchievementType(req.WinType); achievementType != "" {
			if err := l.svcCtx.MatchModel.SaveAchievementWithTx(tx, match.Id, achievementType, 1); err != nil {
				return err
			}
		}
		return l.svcCtx.MatchModel.CreateActionWithRevisionWithTx(tx, action, serverRevision)
	}); err != nil {
		if isRetryableMatchWriteError(err) {
			replayState, replayErr := reloadMatchWriteReplayState(l.svcCtx, userId, req.MatchId, req.ClientActionId)
			if replayErr != nil {
				l.Logger.Errorf("重载写入快照失败: matchId=%d, clientActionId=%s, err=%v", req.MatchId, req.ClientActionId, replayErr)
				return &types.EndRoundResp{Success: false, Accepted: false}, nil
			}
			scoreView := buildMatchWriteScoreView(userId, replayState.Match)
			if replayState.ExistingAction != nil {
				return &types.EndRoundResp{
					Accepted:                  true,
					Success:                   true,
					ClientActionId:            req.ClientActionId,
					ServerRevision:            replayState.View.Snapshot.ServerRevision,
					Snapshot:                  replayState.View.Snapshot,
					RoundNo:                   replayState.ExistingAction.RoundNo,
					CurrentFrameStarted:       replayState.Match.CurrentFrameStarted,
					CurrentFrameMyScore:       scoreView.CurrentFrameMyScore,
					CurrentFrameOpponentScore: scoreView.CurrentFrameOpponentScore,
					MyScore:                   scoreView.MyScore,
					OpponentScore:             scoreView.OpponentScore,
				}, nil
			}
			return &types.EndRoundResp{
				Accepted:                  false,
				Success:                   false,
				ClientActionId:            req.ClientActionId,
				ServerRevision:            replayState.View.Snapshot.ServerRevision,
				Snapshot:                  replayState.View.Snapshot,
				RoundNo:                   int(replayState.View.CompletedRoundCount),
				CurrentFrameStarted:       replayState.Match.CurrentFrameStarted,
				CurrentFrameMyScore:       scoreView.CurrentFrameMyScore,
				CurrentFrameOpponentScore: scoreView.CurrentFrameOpponentScore,
				MyScore:                   scoreView.MyScore,
				OpponentScore:             scoreView.OpponentScore,
			}, nil
		}
		l.Logger.Errorf("结束单局事务失败: %v", err)
		return &types.EndRoundResp{Success: false}, nil
	}
	view, stateErr := loadMatchWriteState(l.svcCtx, userId, match)
	if stateErr != nil {
		l.Logger.Errorf("加载写入快照失败: matchId=%d, clientActionId=%s, err=%v", match.Id, req.ClientActionId, stateErr)
		return &types.EndRoundResp{Success: false, Accepted: false}, nil
	}

	// 先立即推送 WebSocket 消息（优化响应延迟）
	actor := "me"
	if req.Winner == 2 {
		actor = "opponent"
	}
	if ws.GlobalHub != nil {
		currentRoundForClient := newRoundNo
		if match.GameType == 1 && !match.CurrentFrameStarted {
			currentRoundForClient = newRoundNo + 1
		}
		l.Logger.Infof("广播单局结束: matchId=%d, roundNo=%d, winner=%d, player1=%d, player2=%d",
			match.Id, newRoundNo, req.Winner, match.MyScore, match.OpponentScore)
		ws.GlobalHub.BroadcastToMatch(match.Id, &ws.Message{
			Type: "round_end",
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
				CurrentRound:                  currentRoundForClient,
				TotalRounds:                   newRoundNo,
				RoundNumber:                   newRoundNo,
				RoundPlayer1Score:             round.MyScore,
				RoundPlayer2Score:             round.OpponentScore,
				Winner:                        req.Winner,
				ActionType:                    "win",
				Actor:                         actor,
				RedBallCount:                  0,
				SnookerClearanceStarted:       false,
				SnookerClearedColors:          []int{},
				SnookerExpectedClearanceScore: 0,
				SnookerClearanceCompleted:     false,
				Status:                        match.Status,
			},
		})
	}

	l.Logger.Infof("用户 %d 对局 %d 第 %d 局结束，获胜方: %d", userId, match.Id, newRoundNo, req.Winner)

	scoreView := buildMatchWriteScoreView(userId, match)
	return &types.EndRoundResp{
		Accepted:                  true,
		Success:                   true,
		ClientActionId:            req.ClientActionId,
		ServerRevision:            view.Snapshot.ServerRevision,
		Snapshot:                  view.Snapshot,
		RoundNo:                   newRoundNo,
		CurrentFrameStarted:       match.CurrentFrameStarted,
		CurrentFrameMyScore:       scoreView.CurrentFrameMyScore,
		CurrentFrameOpponentScore: scoreView.CurrentFrameOpponentScore,
		MyScore:                   scoreView.MyScore,
		OpponentScore:             scoreView.OpponentScore,
	}, nil
}
