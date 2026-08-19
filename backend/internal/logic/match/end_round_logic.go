package match

import (
	"context"
	"time"

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
		svcCtx: svcCtx.WithContext(ctx),
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
	if _, authorityErr := validateMatchWriteAuthority(match, userId); authorityErr != nil {
		if authorityErr == errMatchViewerNotParticipant {
			l.Logger.Errorf("用户不是对局参与者: matchUserId=%d, matchOpponentId=%v, refereeUserId=%v, currentUserId=%d",
				match.UserId, match.OpponentId, match.RefereeUserId, userId)
			return &types.EndRoundResp{Success: false}, nil
		}
		view, stateErr := loadMatchWriteState(l.svcCtx, userId, match)
		if stateErr != nil {
			l.Logger.Errorf("加载权限拒绝快照失败: matchId=%d, userId=%d, err=%v", match.Id, userId, stateErr)
			return &types.EndRoundResp{Success: false, Accepted: false}, nil
		}
		scoreView := buildMatchWriteScoreView(userId, match)
		return &types.EndRoundResp{
			Success:                   false,
			Accepted:                  false,
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
	if isSnookerV2Match(match) {
		view, stateErr := loadMatchWriteState(l.svcCtx, userId, match)
		if stateErr != nil {
			return &types.EndRoundResp{Success: false, Accepted: false}, nil
		}
		return &types.EndRoundResp{
			Success:        false,
			Accepted:       false,
			ServerRevision: view.Snapshot.ServerRevision,
			Snapshot:       view.Snapshot,
		}, nil
	}
	if req.ClientActionId == "" {
		return &types.EndRoundResp{
			Success:  false,
			Accepted: false,
		}, nil
	}
	if model.IsFlexiblePoolMatch(match) {
		return l.endFlexiblePoolRound(req, userId)
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
			if err := l.svcCtx.MatchModel.SaveAchievementWithTx(tx, match.Id, achievementType, 1, req.Winner); err != nil {
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
				MatchFormat:                   view.Snapshot.MatchFormat,
				TargetWins:                    view.Snapshot.TargetWins,
				CanChangeMatchFormat:          view.Snapshot.CanChangeMatchFormat,
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

type flexiblePoolRoundResult struct {
	Match                *model.Match
	Round                *model.MatchRound
	ExistingAction       *model.MatchAction
	FailureMessage       string
	FinishReq            *types.FinishMatchReq
	FinishedNow          bool
	FinishRequested      bool
	Result               int
	CompetitiveRevisions map[int64]int64
	SeasonID             int64
}

func (l *EndRoundLogic) endFlexiblePoolRound(req *types.EndRoundReq, userID int64) (*types.EndRoundResp, error) {
	result := flexiblePoolRoundResult{}
	err := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		locked, findErr := l.svcCtx.MatchModel.FindByIdForUpdateWithTx(tx, req.MatchId)
		if findErr != nil {
			return findErr
		}
		result.Match = locked
		if locked == nil {
			result.FailureMessage = "对局不存在"
			return nil
		}
		if _, authorityErr := validateMatchWriteAuthority(locked, userID); authorityErr != nil {
			result.FailureMessage = authorityErr.Error()
			return nil
		}
		if !model.IsFlexiblePoolMatch(locked) {
			result.FailureMessage = "当前对局不支持逐局赛制"
			return nil
		}
		if existing, findErr := l.svcCtx.MatchModel.FindActionByClientActionIDWithTx(tx, locked.Id, req.ClientActionId); findErr != nil {
			return findErr
		} else if existing != nil {
			if existing.ActionType != "win" || existing.Actor != req.Winner || existing.BaseRevision != req.BaseRevision || existing.ScoreChange != 1 {
				result.FailureMessage = "操作幂等键已被其他操作使用"
				return nil
			}
			result.ExistingAction = existing
			return nil
		}
		if err := validateMatchActionMeta(matchActionMeta{
			ClientActionID: req.ClientActionId,
			BaseRevision:   req.BaseRevision,
		}, locked.SyncRevision); err != nil {
			result.FailureMessage = err.Error()
			return nil
		}
		if locked.Status != 1 || model.NormalizeFinishState(locked.FinishState) != model.FinishStateNone {
			result.FailureMessage = "当前对局不可记分"
			return nil
		}
		if req.Winner != 1 && req.Winner != 2 {
			result.FailureMessage = "局胜方只能是选手1或选手2"
			return nil
		}
		if req.Score != 1 {
			result.FailureMessage = "灵活赛制每局只能计1胜"
			return nil
		}
		roundCount, countErr := l.svcCtx.MatchModel.GetRoundCountWithTx(tx, locked.Id)
		if countErr != nil {
			return countErr
		}
		if poolMatchFormatLimitReached(locked) || roundCount >= model.PoolMatchMaxCompletedRounds {
			result.FailureMessage = "当前赛制已达到结束条件"
			return nil
		}

		if req.Winner == 1 {
			locked.MyScore++
		} else {
			locked.OpponentScore++
		}
		newRoundNo := int(roundCount) + 1
		round := &model.MatchRound{
			MatchId:       locked.Id,
			RoundNo:       newRoundNo,
			MyScore:       locked.MyScore,
			OpponentScore: locked.OpponentScore,
			Winner:        &req.Winner,
			WinType:       req.WinType,
		}
		extraData := `{"win_type":"` + req.WinType + `"}`
		action := &model.MatchAction{
			MatchId:        locked.Id,
			RoundNo:        newRoundNo,
			ActionType:     "win",
			Actor:          req.Winner,
			ScoreChange:    1,
			ClientActionId: stringPointer(req.ClientActionId),
			BaseRevision:   req.BaseRevision,
			ExtraData:      &extraData,
		}
		revision, revisionErr := l.svcCtx.MatchModel.BumpMatchRevisionWithTx(tx, locked)
		if revisionErr != nil {
			return revisionErr
		}
		if err := l.svcCtx.MatchModel.CreateRoundWithTx(tx, round); err != nil {
			return err
		}
		if achievementType := normalizeStoredAchievementType(req.WinType); achievementType != "" {
			if err := l.svcCtx.MatchModel.SaveAchievementWithTx(tx, locked.Id, achievementType, 1, req.Winner); err != nil {
				return err
			}
		}
		if err := l.svcCtx.MatchModel.CreateActionWithRevisionWithTx(tx, action, revision); err != nil {
			return err
		}
		result.Round = round

		if poolMatchFormatLimitReached(locked) {
			if locked.FinishConfirmationRequired && model.NormalizeMatchMode(locked.MatchMode) == model.MatchModeRanked && locked.RefereeUserId == nil {
				requestRevision := locked.SyncRevision
				locked.FinishState = model.FinishStatePendingConfirmation
				locked.FinishRequestedBy = &userID
				now := time.Now()
				locked.FinishRequestedAt = &now
				locked.FinishRequestRevision = requestRevision + 1
				revision, err := l.svcCtx.MatchModel.BumpMatchRevisionWithTx(tx, locked)
				if err != nil {
					return err
				}
				if err := l.svcCtx.MatchModel.CreateActionWithRevisionWithTx(tx, &model.MatchAction{
					MatchId:        locked.Id,
					RoundNo:        0,
					ActionType:     "finish_request",
					Actor:          resolveFinishActionActor(locked, userID),
					ClientActionId: stringPointer(autoPoolFinishRequestActionID(req.ClientActionId)),
					BaseRevision:   requestRevision,
				}, revision); err != nil {
					return err
				}
				result.FinishRequested = true
			} else {
				finishReq := &types.FinishMatchReq{
					MatchId:        locked.Id,
					ClientActionId: autoPoolFinishActionID(req.ClientActionId),
					BaseRevision:   locked.SyncRevision,
				}
				settlement, err := NewFinishMatchLogic(l.ctx, l.svcCtx).settleMatchWithTx(tx, locked, userID, finishReq)
				if err != nil {
					return err
				}
				result.FinishedNow = true
				result.FinishReq = finishReq
				result.Result = settlement.Result
				result.CompetitiveRevisions = settlement.CompetitiveRevisions
				result.SeasonID = settlement.SeasonID
			}
		}
		return nil
	})
	if err != nil {
		l.Logger.Errorf("灵活赛制结束单局事务失败: matchId=%d err=%v", req.MatchId, err)
		return &types.EndRoundResp{Success: false, Accepted: false}, nil
	}

	fresh, _ := l.svcCtx.MatchModel.FindById(req.MatchId)
	if fresh == nil {
		return &types.EndRoundResp{Success: false, Accepted: false}, nil
	}
	view, stateErr := loadMatchWriteState(l.svcCtx, userID, fresh)
	if stateErr != nil {
		return &types.EndRoundResp{Success: false, Accepted: false}, nil
	}
	scoreView := buildMatchWriteScoreView(userID, fresh)
	if result.FailureMessage != "" {
		return &types.EndRoundResp{
			Accepted:                  false,
			Success:                   false,
			ClientActionId:            req.ClientActionId,
			ServerRevision:            view.Snapshot.ServerRevision,
			Snapshot:                  view.Snapshot,
			RoundNo:                   int(view.CompletedRoundCount),
			CurrentFrameStarted:       fresh.CurrentFrameStarted,
			CurrentFrameMyScore:       scoreView.CurrentFrameMyScore,
			CurrentFrameOpponentScore: scoreView.CurrentFrameOpponentScore,
			MyScore:                   scoreView.MyScore,
			OpponentScore:             scoreView.OpponentScore,
		}, nil
	}
	roundNo := int(view.CompletedRoundCount)
	if result.ExistingAction != nil {
		roundNo = result.ExistingAction.RoundNo
	}
	if result.Round != nil {
		roundNo = result.Round.RoundNo
		broadcastFlexiblePoolRoundEnd(fresh, view, result.Round, req.Winner)
	}
	if result.ExistingAction != nil && fresh.Status == 2 && fresh.Result != nil {
		finishLogic := NewFinishMatchLogic(l.ctx, l.svcCtx)
		if _, err := finishLogic.finishMatchPostCommit(&types.FinishMatchReq{
			MatchId:        fresh.Id,
			ClientActionId: autoPoolFinishActionID(req.ClientActionId),
			BaseRevision:   result.ExistingAction.ServerRevision,
		}, userID, fresh, *fresh.Result, nil, 0); err != nil {
			finishLogic.Logger.Errorf("灵活赛制幂等重放补偿失败: matchId=%d err=%v", fresh.Id, err)
		}
	}
	if result.FinishedNow && result.FinishReq != nil {
		finishLogic := NewFinishMatchLogic(l.ctx, l.svcCtx)
		if _, err := finishLogic.finishMatchPostCommit(result.FinishReq, userID, fresh, result.Result, result.CompetitiveRevisions, result.SeasonID); err != nil {
			finishLogic.Logger.Errorf("灵活赛制自动结束后处理失败: matchId=%d err=%v", fresh.Id, err)
		}
	}
	if result.FinishRequested {
		broadcastFinishActionState(l.svcCtx, fresh, "match_finish_request")
	}
	return &types.EndRoundResp{
		Accepted:                  true,
		Success:                   true,
		ClientActionId:            req.ClientActionId,
		ServerRevision:            view.Snapshot.ServerRevision,
		Snapshot:                  view.Snapshot,
		RoundNo:                   roundNo,
		CurrentFrameStarted:       fresh.CurrentFrameStarted,
		CurrentFrameMyScore:       scoreView.CurrentFrameMyScore,
		CurrentFrameOpponentScore: scoreView.CurrentFrameOpponentScore,
		MyScore:                   scoreView.MyScore,
		OpponentScore:             scoreView.OpponentScore,
	}, nil
}

func broadcastFlexiblePoolRoundEnd(match *model.Match, view matchWriteState, round *model.MatchRound, winner int) {
	if ws.GlobalHub == nil || match == nil || round == nil {
		return
	}
	actor := "me"
	if winner == 2 {
		actor = "opponent"
	}
	ws.GlobalHub.BroadcastToMatch(match.Id, &ws.Message{
		Type: "round_end",
		Data: ws.ScoreUpdateData{
			MatchId:              match.Id,
			ServerRevision:       view.Snapshot.ServerRevision,
			MyScore:              match.MyScore,
			OpponentScore:        match.OpponentScore,
			Player1Score:         match.MyScore,
			Player2Score:         match.OpponentScore,
			CurrentFrameStarted:  match.CurrentFrameStarted,
			CurrentRound:         view.Snapshot.CurrentRound,
			TotalRounds:          view.Snapshot.TotalRounds,
			RoundNumber:          round.RoundNo,
			RoundPlayer1Score:    round.MyScore,
			RoundPlayer2Score:    round.OpponentScore,
			Winner:               winner,
			ActionType:           "win",
			Actor:                actor,
			MatchFormat:          view.Snapshot.MatchFormat,
			TargetWins:           view.Snapshot.TargetWins,
			CanChangeMatchFormat: view.Snapshot.CanChangeMatchFormat,
			Status:               match.Status,
		},
	})
}

func autoPoolFinishActionID(clientActionID string) string {
	return clientActionID + ":pool_finish"
}

func autoPoolFinishRequestActionID(clientActionID string) string {
	return clientActionID + ":pool_finish_request"
}
