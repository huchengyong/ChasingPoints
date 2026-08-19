package match

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

type StartNextRoundLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 开始下一局
func NewStartNextRoundLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StartNextRoundLogic {
	return &StartNextRoundLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *StartNextRoundLogic) StartNextRound(req *types.StartNextRoundReq) (resp *types.StartNextRoundResp, err error) {
	// 获取用户ID
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.StartNextRoundResp{Success: false}, nil
	}

	// 查询对局
	match, err := l.svcCtx.MatchModel.FindById(req.MatchId)
	if err != nil || match == nil {
		l.Logger.Errorf("对局不存在: %v", err)
		return &types.StartNextRoundResp{Success: false}, nil
	}

	// 验证用户权限
	if _, authorityErr := validateMatchWriteAuthority(match, userId); authorityErr != nil {
		if authorityErr == errMatchViewerNotParticipant {
			l.Logger.Errorf("用户不是对局参与者: matchUserId=%d, matchOpponentId=%v, refereeUserId=%v, currentUserId=%d",
				match.UserId, match.OpponentId, match.RefereeUserId, userId)
			return &types.StartNextRoundResp{Success: false}, nil
		}
		view, stateErr := loadMatchWriteState(l.svcCtx, userId, match)
		if stateErr != nil {
			l.Logger.Errorf("加载权限拒绝快照失败: matchId=%d, userId=%d, err=%v", match.Id, userId, stateErr)
			return &types.StartNextRoundResp{Success: false, Accepted: false}, nil
		}
		scoreView := buildMatchWriteScoreView(userId, match)
		return &types.StartNextRoundResp{
			Success:                   false,
			Accepted:                  false,
			ServerRevision:            view.Snapshot.ServerRevision,
			Snapshot:                  view.Snapshot,
			RoundNo:                   int(view.CompletedRoundCount) + 1,
			CurrentFrameStarted:       match.CurrentFrameStarted,
			CurrentFrameMyScore:       scoreView.CurrentFrameMyScore,
			CurrentFrameOpponentScore: scoreView.CurrentFrameOpponentScore,
			MyScore:                   scoreView.MyScore,
			OpponentScore:             scoreView.OpponentScore,
		}, nil
	}
	if req.ClientActionId == "" {
		return &types.StartNextRoundResp{
			Success:  false,
			Accepted: false,
		}, nil
	}
	existingAction, err := l.svcCtx.MatchModel.FindActionByClientActionID(match.Id, req.ClientActionId)
	if err != nil {
		l.Logger.Errorf("查询幂等操作失败: matchId=%d, clientActionId=%s, err=%v", match.Id, req.ClientActionId, err)
		return &types.StartNextRoundResp{Success: false, Accepted: false}, nil
	}
	if existingAction != nil {
		view, stateErr := loadMatchWriteState(l.svcCtx, userId, match)
		if stateErr != nil {
			l.Logger.Errorf("加载重放快照失败: matchId=%d, clientActionId=%s, err=%v", match.Id, req.ClientActionId, stateErr)
			return &types.StartNextRoundResp{Success: false, Accepted: false}, nil
		}
		scoreView := buildMatchWriteScoreView(userId, match)
		return &types.StartNextRoundResp{
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
			return &types.StartNextRoundResp{Success: false, Accepted: false}, nil
		}
		scoreView := buildMatchWriteScoreView(userId, match)
		return &types.StartNextRoundResp{
			Success:                   false,
			Accepted:                  false,
			ClientActionId:            req.ClientActionId,
			ServerRevision:            view.Snapshot.ServerRevision,
			Snapshot:                  view.Snapshot,
			RoundNo:                   int(view.CompletedRoundCount) + 1,
			CurrentFrameStarted:       match.CurrentFrameStarted,
			CurrentFrameMyScore:       scoreView.CurrentFrameMyScore,
			CurrentFrameOpponentScore: scoreView.CurrentFrameOpponentScore,
			MyScore:                   scoreView.MyScore,
			OpponentScore:             scoreView.OpponentScore,
		}, nil
	}

	// 验证对局状态
	if match.Status != 1 {
		l.Logger.Errorf("对局状态不是进行中: matchId=%d, status=%d", match.Id, match.Status)
		return &types.StartNextRoundResp{Success: false}, nil
	}

	// 获取当前局数
	roundCount, _ := l.svcCtx.MatchModel.GetRoundCount(match.Id)
	newRoundNo := int(roundCount) + 1

	if match.GameType == 1 {
		if snookerFormatLimitReached(match) {
			l.Logger.Errorf("斯诺克赛制已达到结束条件: matchId=%d", match.Id)
			return &types.StartNextRoundResp{Success: false, Accepted: false}, nil
		}
		if match.CurrentFrameStarted {
			l.Logger.Errorf("斯诺克当前局尚未结束: matchId=%d, roundNo=%d", match.Id, newRoundNo)
			return &types.StartNextRoundResp{Success: false}, nil
		}
		if isSnookerV2Match(match) {
			state, stateErr := loadSnookerStateForMatch(l.svcCtx, match, int(roundCount))
			if stateErr != nil || !state.FrameEnded {
				return &types.StartNextRoundResp{Success: false}, nil
			}
		}
		match.CurrentFrameStarted = true
		match.CurrentFrameMyScore = 0
		match.CurrentFrameOpponentScore = 0
	}

	// 记录操作日志
	action := &model.MatchAction{
		MatchId:        match.Id,
		RoundNo:        newRoundNo,
		ActionType:     "round_start",
		Actor:          0, // 系统操作
		ScoreChange:    0,
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
		return l.svcCtx.MatchModel.CreateActionWithRevisionWithTx(tx, action, serverRevision)
	}); err != nil {
		if isRetryableMatchWriteError(err) {
			replayState, replayErr := reloadMatchWriteReplayState(l.svcCtx, userId, req.MatchId, req.ClientActionId)
			if replayErr != nil {
				l.Logger.Errorf("重载写入快照失败: matchId=%d, clientActionId=%s, err=%v", req.MatchId, req.ClientActionId, replayErr)
				return &types.StartNextRoundResp{Success: false, Accepted: false}, nil
			}
			scoreView := buildMatchWriteScoreView(userId, replayState.Match)
			if replayState.ExistingAction != nil {
				return &types.StartNextRoundResp{
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
			return &types.StartNextRoundResp{
				Accepted:                  false,
				Success:                   false,
				ClientActionId:            req.ClientActionId,
				ServerRevision:            replayState.View.Snapshot.ServerRevision,
				Snapshot:                  replayState.View.Snapshot,
				RoundNo:                   int(replayState.View.CompletedRoundCount) + 1,
				CurrentFrameStarted:       replayState.Match.CurrentFrameStarted,
				CurrentFrameMyScore:       scoreView.CurrentFrameMyScore,
				CurrentFrameOpponentScore: scoreView.CurrentFrameOpponentScore,
				MyScore:                   scoreView.MyScore,
				OpponentScore:             scoreView.OpponentScore,
			}, nil
		}
		l.Logger.Errorf("开始下一局事务失败: %v", err)
		return &types.StartNextRoundResp{Success: false}, nil
	}
	view, stateErr := loadMatchWriteState(l.svcCtx, userId, match)
	if stateErr != nil {
		l.Logger.Errorf("加载写入快照失败: matchId=%d, clientActionId=%s, err=%v", match.Id, req.ClientActionId, stateErr)
		return &types.StartNextRoundResp{Success: false, Accepted: false}, nil
	}

	// 推送 WebSocket 消息
	if ws.GlobalHub != nil {
		l.Logger.Infof("广播下一局开始: matchId=%d, currentRound=%d, totalRounds=%d",
			match.Id, newRoundNo, int(roundCount))
		ws.GlobalHub.BroadcastToMatch(match.Id, &ws.Message{
			Type: "round_start",
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
				CurrentRound:                  newRoundNo,
				TotalRounds:                   int(roundCount),
				ActionType:                    "round_start",
				Actor:                         "",
				RedBallCount:                  0,
				SnookerClearanceStarted:       false,
				SnookerClearedColors:          []int{},
				SnookerExpectedClearanceScore: 0,
				SnookerClearanceCompleted:     view.SnookerState.ClearanceCompleted,
				SnookerRulesVersion:           match.SnookerRulesVersion,
				BestOfFrames:                  match.BestOfFrames,
				SnookerFormat:                 match.SnookerFormat,
				SnookerTargetWins:             match.SnookerTargetWins,
				MatchFormat:                   view.Snapshot.MatchFormat,
				TargetWins:                    view.Snapshot.TargetWins,
				CanChangeMatchFormat:          view.Snapshot.CanChangeMatchFormat,
				StartingActor:                 match.StartingActor,
				SnookerPhase:                  view.SnookerState.Phase,
				SnookerBallOn:                 view.SnookerState.BallOn,
				SnookerStriker:                view.SnookerState.Striker,
				SnookerVisitNo:                view.SnookerState.VisitNo,
				SnookerCurrentBreak:           view.SnookerState.CurrentBreak,
				SnookerRedsRemaining:          view.SnookerState.RedsRemaining,
				SnookerFreeBallAvailable:      view.SnookerState.FreeBallAvailable,
				SnookerCueBallInHand:          view.SnookerState.CueBallInHand,
				SnookerMissWarningActive:      view.SnookerState.MissWarningActive,
				SnookerRespottedBlackPending:  view.SnookerState.Phase == model.SnookerPhaseRespottedBlackPending,
				SnookerPendingConcessionActor: view.SnookerState.PendingConcessionActor,
				SnookerPendingConcessionScope: view.SnookerState.PendingConcessionScope,
				SnookerFrameEndReason:         view.SnookerState.FrameEndReason,
				Status:                        match.Status,
			},
		})
	}

	l.Logger.Infof("用户 %d 对局 %d 开始第 %d 局", userId, match.Id, newRoundNo)

	scoreView := buildMatchWriteScoreView(userId, match)
	return &types.StartNextRoundResp{
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
