package match

import (
	"context"
	"errors"
	"fmt"
	"time"

	"chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/pkg/ws"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"gorm.io/gorm"
)

const finishRequestTTL = model.FinishRequestTTL

var (
	errFinishActionInvalid        = errors.New("当前无法处理结束请求")
	errFinishActionExpired        = errors.New("结束请求已过期")
	errFinishActionNotParticipant = errors.New("用户不是对局参与者")
)

func shouldRequestRankedFinish(match *model.Match, userId int64) bool {
	if match == nil || match.Status != 1 || userId <= 0 || !match.FinishConfirmationRequired || model.NormalizeMatchMode(match.MatchMode) != model.MatchModeRanked {
		return false
	}
	if match.RefereeUserId != nil && *match.RefereeUserId > 0 {
		return false
	}
	return match.UserId == userId || (match.OpponentId != nil && *match.OpponentId == userId)
}

func buildMatchWriteScoreViewFromSnapshot(snapshot types.MatchSyncSnapshot) matchWriteScoreView {
	return matchWriteScoreView{
		MyScore:                   snapshot.MyScore,
		OpponentScore:             snapshot.OpponentScore,
		CurrentFrameMyScore:       snapshot.CurrentFrameMyScore,
		CurrentFrameOpponentScore: snapshot.CurrentFrameOpponentScore,
	}
}

func (l *RequestFinishMatchLogic) request(req *types.FinishMatchActionReq) (*types.FinishMatchActionResp, error) {
	return handleFinishAction(l.ctx, l.svcCtx, req, "request")
}

func (l *ConfirmFinishMatchLogic) confirm(req *types.FinishMatchActionReq) (*types.FinishMatchActionResp, error) {
	return handleConfirmedFinishAction(l.ctx, l.svcCtx, req)
}

func (l *DisputeFinishMatchLogic) dispute(req *types.FinishMatchActionReq) (*types.FinishMatchActionResp, error) {
	return handleFinishAction(l.ctx, l.svcCtx, req, "dispute")
}

func (l *WithdrawFinishMatchLogic) withdraw(req *types.FinishMatchActionReq) (*types.FinishMatchActionResp, error) {
	return handleFinishAction(l.ctx, l.svcCtx, req, "withdraw")
}

func handleConfirmedFinishAction(ctx context.Context, svcCtx *svc.ServiceContext, req *types.FinishMatchActionReq) (*types.FinishMatchActionResp, error) {
	if req == nil || req.MatchId <= 0 || req.ClientActionId == "" || svcCtx == nil || svcCtx.MatchModel == nil || svcCtx.DB == nil {
		return &types.FinishMatchActionResp{Success: false, Accepted: false, Message: "请求参数无效"}, nil
	}
	userId, err := utils.GetUserIDFromCtx(ctx)
	if err != nil {
		return &types.FinishMatchActionResp{Success: false, Accepted: false, Message: "获取用户信息失败"}, nil
	}

	current, findErr := svcCtx.MatchModel.FindById(req.MatchId)
	if findErr != nil || current == nil {
		return &types.FinishMatchActionResp{Success: false, Accepted: false, ClientActionId: req.ClientActionId, Message: "对局不存在"}, nil
	}
	if isFinishRequestExpired(current, time.Now()) {
		return buildFinishActionFailureResponse(svcCtx, userId, effectiveMatchForRead(current, time.Now()), req.ClientActionId, "结束请求已过期，对局已恢复进行中"), nil
	}

	var match *model.Match
	var settlement finishMatchSettlement
	replayed := false
	err = svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		locked, findErr := svcCtx.MatchModel.FindByIdForUpdateWithTx(tx, req.MatchId)
		if findErr != nil {
			return findErr
		}
		if locked == nil {
			return gorm.ErrRecordNotFound
		}
		match = locked
		capabilities := resolveMatchViewerCapabilities(locked, userId)
		if capabilities.ViewerRole == matchViewerRoleUnknown {
			return errFinishActionNotParticipant
		}
		existing, findErr := svcCtx.MatchModel.FindActionByClientActionIDWithTx(tx, locked.Id, req.ClientActionId)
		if findErr != nil {
			return findErr
		}
		if existing != nil {
			if !isSameFinishAction(existing, locked, userId, req, "confirm") || locked.Status != 2 {
				return errFinishActionInvalid
			}
			replayed = true
			return nil
		}
		if err := validateMatchActionMeta(matchActionMeta{ClientActionID: req.ClientActionId, BaseRevision: req.BaseRevision}, locked.SyncRevision); err != nil {
			return err
		}
		if locked.Status != 1 || !locked.FinishConfirmationRequired || model.NormalizeMatchMode(locked.MatchMode) != model.MatchModeRanked || locked.RefereeUserId != nil {
			return errFinishActionInvalid
		}
		if isFinishRequestExpired(locked, time.Now()) {
			return errFinishActionExpired
		}
		requestedBy := resolveFinishRequestedBy(locked)
		if locked.FinishState != model.FinishStatePendingConfirmation || !capabilities.CanConfirmFinish || requestedBy == userId {
			return errFinishActionInvalid
		}

		clearFinishRequest(locked)
		confirmRevision, err := svcCtx.MatchModel.BumpMatchRevisionWithTx(tx, locked)
		if err != nil {
			return err
		}
		if err := svcCtx.MatchModel.CreateActionWithRevisionWithTx(tx, &model.MatchAction{
			MatchId:        locked.Id,
			RoundNo:        0,
			ActionType:     "finish_confirm",
			Actor:          resolveFinishActionActor(locked, userId),
			ClientActionId: stringPointer(req.ClientActionId),
			BaseRevision:   req.BaseRevision,
		}, confirmRevision); err != nil {
			return err
		}
		settlement, err = NewFinishMatchLogic(ctx, svcCtx).settleMatchWithCompletionSourceTx(tx, locked, userId, &types.FinishMatchReq{
			MatchId:        req.MatchId,
			ClientActionId: req.ClientActionId + ":settle",
			BaseRevision:   confirmRevision,
		}, model.CompletionSourcePlayerConfirmed)
		return err
	})
	if err != nil {
		message := "当前无法处理结束请求"
		switch {
		case errors.Is(err, errFinishActionNotParticipant):
			message = "你不是本场对局参与者"
		case errors.Is(err, errRevisionConflict):
			message = errRevisionConflict.Error()
		case errors.Is(err, errFinishActionInvalid):
			message = errFinishActionInvalid.Error()
		}
		fresh, _ := svcCtx.MatchModel.FindById(req.MatchId)
		return buildFinishActionFailureResponse(svcCtx, userId, fresh, req.ClientActionId, message), nil
	}
	if match == nil {
		return &types.FinishMatchActionResp{Success: false, Accepted: false, ClientActionId: req.ClientActionId, Message: "当前无法处理结束请求"}, nil
	}

	if replayed {
		NewFinishMatchLogic(ctx, svcCtx).syncAchievementProgressForCompletedMatch(match)
		view, viewErr := loadMatchWriteState(svcCtx, userId, match)
		if viewErr != nil {
			return &types.FinishMatchActionResp{Success: false, Accepted: false, ClientActionId: req.ClientActionId, Message: "加载对局快照失败"}, nil
		}
		return &types.FinishMatchActionResp{
			Accepted:          true,
			Success:           true,
			Action:            "confirm",
			Message:           "已确认结束并完成结算",
			ClientActionId:    req.ClientActionId,
			ServerRevision:    view.Snapshot.ServerRevision,
			FinishState:       view.Snapshot.FinishState,
			FinishRequestedBy: view.Snapshot.FinishRequestedBy,
			Snapshot:          view.Snapshot,
		}, nil
	}

	broadcastFinishActionState(svcCtx, match, "match_finish_confirm")
	finishResp, finishErr := NewFinishMatchLogic(ctx, svcCtx).finishMatchPostCommit(&types.FinishMatchReq{
		MatchId:        req.MatchId,
		ClientActionId: req.ClientActionId,
		BaseRevision:   req.BaseRevision,
	}, userId, match, settlement.Result, settlement.CompetitiveRevisions, settlement.SeasonID)
	if finishErr != nil || finishResp == nil || !finishResp.Success {
		return &types.FinishMatchActionResp{Success: false, Accepted: false, ClientActionId: req.ClientActionId, Message: "加载对局结算结果失败"}, finishErr
	}
	return &types.FinishMatchActionResp{
		Accepted:          true,
		Success:           true,
		Action:            "confirm",
		Message:           "已确认结束并完成结算",
		ClientActionId:    req.ClientActionId,
		ServerRevision:    finishResp.ServerRevision,
		FinishState:       finishResp.Snapshot.FinishState,
		FinishRequestedBy: finishResp.Snapshot.FinishRequestedBy,
		Snapshot:          finishResp.Snapshot,
	}, nil
}

func buildFinishActionFailureResponse(svcCtx *svc.ServiceContext, userId int64, match *model.Match, clientActionId, message string) *types.FinishMatchActionResp {
	resp := &types.FinishMatchActionResp{Success: false, Accepted: false, ClientActionId: clientActionId, Message: message}
	if match == nil {
		return resp
	}
	if view, err := loadMatchWriteState(svcCtx, userId, match); err == nil {
		resp.ServerRevision = view.Snapshot.ServerRevision
		resp.FinishState = view.Snapshot.FinishState
		resp.FinishRequestedBy = view.Snapshot.FinishRequestedBy
		resp.Snapshot = view.Snapshot
	}
	return resp
}

func handleFinishAction(ctx context.Context, svcCtx *svc.ServiceContext, req *types.FinishMatchActionReq, actionType string) (*types.FinishMatchActionResp, error) {
	if req == nil || req.MatchId <= 0 || req.ClientActionId == "" || svcCtx == nil || svcCtx.MatchModel == nil || svcCtx.DB == nil {
		return &types.FinishMatchActionResp{Success: false, Accepted: false, Message: "请求参数无效"}, nil
	}
	userId, err := utils.GetUserIDFromCtx(ctx)
	if err != nil {
		return &types.FinishMatchActionResp{Success: false, Accepted: false, Message: "获取用户信息失败"}, nil
	}
	if current, findErr := svcCtx.MatchModel.FindById(req.MatchId); findErr == nil && current != nil && isSnookerV2Match(current) && actionType == "request" && !snookerNormalFinishEligible(current) {
		return buildFinishActionFailureResponse(svcCtx, userId, current, req.ClientActionId, "当前斯诺克赛制尚未满足正常结束条件"), nil
	}

	var match *model.Match
	var revision int64
	var actionMessage string
	err = svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		locked, findErr := svcCtx.MatchModel.FindByIdForUpdateWithTx(tx, req.MatchId)
		if findErr != nil {
			return findErr
		}
		if locked == nil {
			return gorm.ErrRecordNotFound
		}
		match = locked
		capabilities := resolveMatchViewerCapabilities(locked, userId)
		if capabilities.ViewerRole == matchViewerRoleUnknown {
			return errFinishActionNotParticipant
		}
		if locked.FinishState == model.FinishStatePendingConfirmation && locked.FinishRequestedAt != nil && time.Since(*locked.FinishRequestedAt) >= finishRequestTTL {
			clearFinishRequest(locked)
			revision, err = svcCtx.MatchModel.BumpMatchRevisionWithTx(tx, locked)
			if err != nil {
				return err
			}
			match = locked
			if actionType != "request" {
				return errFinishActionExpired
			}
			revision = 0
		}
		if existing, findErr := svcCtx.MatchModel.FindActionByClientActionIDWithTx(tx, locked.Id, req.ClientActionId); findErr != nil {
			return findErr
		} else if existing != nil {
			if !isSameFinishAction(existing, locked, userId, req, actionType) {
				return errFinishActionInvalid
			}
			revision = locked.SyncRevision
			return nil
		}
		if err := validateMatchActionMeta(matchActionMeta{ClientActionID: req.ClientActionId, BaseRevision: req.BaseRevision}, locked.SyncRevision); err != nil {
			return err
		}

		if locked.Status != 1 || !locked.FinishConfirmationRequired || model.NormalizeMatchMode(locked.MatchMode) != model.MatchModeRanked || locked.RefereeUserId != nil {
			return errFinishActionInvalid
		}
		requestedBy := resolveFinishRequestedBy(locked)
		switch actionType {
		case "request":
			if !capabilities.CanRequestFinish || locked.FinishState == model.FinishStatePendingConfirmation {
				return errFinishActionInvalid
			}
			locked.FinishState = model.FinishStatePendingConfirmation
			locked.FinishRequestedBy = &userId
			now := time.Now()
			locked.FinishRequestedAt = &now
			locked.FinishRequestRevision = locked.SyncRevision + 1
			actionMessage = "已发起结束确认，等待对手处理"
		case "confirm":
			if locked.FinishState != model.FinishStatePendingConfirmation || !capabilities.CanConfirmFinish || requestedBy == userId {
				return errFinishActionInvalid
			}
			clearFinishRequest(locked)
			actionMessage = "已确认结束"
		case "dispute":
			if locked.FinishState != model.FinishStatePendingConfirmation || !capabilities.CanDisputeFinish || requestedBy == userId {
				return errFinishActionInvalid
			}
			clearFinishRequest(locked)
			actionMessage = "已提出异议，对局恢复进行中"
		case "withdraw":
			if locked.FinishState != model.FinishStatePendingConfirmation || !capabilities.CanWithdrawFinish || requestedBy != userId {
				return errFinishActionInvalid
			}
			clearFinishRequest(locked)
			actionMessage = "已撤回结束请求"
		default:
			return errFinishActionInvalid
		}

		if revision == 0 {
			revision, err = svcCtx.MatchModel.BumpMatchRevisionWithTx(tx, locked)
			if err != nil {
				return err
			}
		}
		action := &model.MatchAction{
			MatchId:        locked.Id,
			RoundNo:        0,
			ActionType:     "finish_" + actionType,
			Actor:          resolveFinishActionActor(locked, userId),
			ClientActionId: stringPointer(req.ClientActionId),
			BaseRevision:   req.BaseRevision,
		}
		return svcCtx.MatchModel.CreateActionWithRevisionWithTx(tx, action, revision)
	})
	if err != nil {
		message := "当前无法处理结束请求"
		switch {
		case errors.Is(err, errFinishActionExpired):
			message = "结束请求已过期，对局已恢复进行中"
		case errors.Is(err, errFinishActionNotParticipant):
			message = "你不是本场对局参与者"
		case errors.Is(err, errRevisionConflict):
			message = errRevisionConflict.Error()
		case errors.Is(err, errFinishActionInvalid):
			message = errFinishActionInvalid.Error()
		}
		if match == nil {
			return &types.FinishMatchActionResp{Success: false, Accepted: false, Message: message}, nil
		}
		view, _ := loadMatchWriteState(svcCtx, userId, match)
		return &types.FinishMatchActionResp{Success: false, Accepted: false, Message: message, ClientActionId: req.ClientActionId, ServerRevision: view.Snapshot.ServerRevision, FinishState: view.Snapshot.FinishState, FinishRequestedBy: view.Snapshot.FinishRequestedBy, Snapshot: view.Snapshot}, nil
	}

	view, viewErr := loadMatchWriteState(svcCtx, userId, match)
	if viewErr != nil {
		return &types.FinishMatchActionResp{Success: false, Accepted: false, Message: "加载对局快照失败"}, nil
	}
	if recipientId := resolveFinishNotificationRecipient(match, userId); actionType == "request" && recipientId > 0 {
		logic.DispatchNotification(svcCtx, logic.NotificationDispatchInput{
			UserId:      recipientId,
			Type:        "match_finish_confirmation",
			Title:       "请确认对局结果",
			Content:     "对手已发起排位结束确认，请核对比分后处理",
			Data:        buildNotificationPayload("/subPages/match/playing", match.Id, 0),
			PushTitle:   "请确认对局结果",
			PushContent: "请打开对局处理结束确认",
			PushData:    map[string]interface{}{"url": "/subPages/match/playing?match_id=" + formatMatchID(match.Id)},
			WSCategory:  "match_finish_confirmation",
		})
	}
	if ws.GlobalHub != nil {
		broadcastFinishActionState(svcCtx, match, "match_finish_"+actionType)
	}
	return &types.FinishMatchActionResp{
		Accepted:          true,
		Success:           true,
		Action:            actionType,
		Message:           actionMessage,
		ClientActionId:    req.ClientActionId,
		ServerRevision:    view.Snapshot.ServerRevision,
		FinishState:       view.Snapshot.FinishState,
		FinishRequestedBy: view.Snapshot.FinishRequestedBy,
		Snapshot:          view.Snapshot,
	}, nil
}

func resolveFinishNotificationRecipient(match *model.Match, requesterId int64) int64 {
	if match == nil || requesterId <= 0 || match.OpponentId == nil || *match.OpponentId <= 0 {
		return 0
	}
	if requesterId == match.UserId {
		return *match.OpponentId
	}
	if requesterId == *match.OpponentId {
		return match.UserId
	}
	return 0
}

func clearFinishRequest(match *model.Match) {
	if match == nil {
		return
	}
	match.FinishState = model.FinishStateNone
	match.FinishRequestedBy = nil
	match.FinishRequestedAt = nil
	match.FinishRequestRevision = 0
}

func isSameFinishAction(existing *model.MatchAction, match *model.Match, userId int64, req *types.FinishMatchActionReq, actionType string) bool {
	if existing == nil || match == nil || req == nil {
		return false
	}
	return existing.ActionType == "finish_"+actionType &&
		existing.Actor == resolveFinishActionActor(match, userId) &&
		existing.BaseRevision == req.BaseRevision
}

func broadcastFinishActionState(svcCtx *svc.ServiceContext, match *model.Match, messageType string) {
	if svcCtx == nil || match == nil || ws.GlobalHub == nil {
		return
	}
	userIds := []int64{0, match.UserId}
	if match.OpponentId != nil && *match.OpponentId > 0 {
		userIds = append(userIds, *match.OpponentId)
	}
	if match.RefereeUserId != nil && *match.RefereeUserId > 0 {
		userIds = append(userIds, *match.RefereeUserId)
	}
	messages := make(map[int64]*ws.Message, len(userIds))
	for _, userId := range userIds {
		if _, exists := messages[userId]; exists {
			continue
		}
		view, err := loadMatchWriteState(svcCtx, userId, match)
		if err != nil {
			continue
		}
		messages[userId] = &ws.Message{Type: messageType, Data: map[string]interface{}{
			"match_id":            match.Id,
			"server_revision":     view.Snapshot.ServerRevision,
			"finish_state":        view.Snapshot.FinishState,
			"finish_requested_by": view.Snapshot.FinishRequestedBy,
			"snapshot":            view.Snapshot,
		}}
	}
	ws.GlobalHub.BroadcastToMatchForUsers(match.Id, messages)
}

func resolveFinishActionActor(match *model.Match, userId int64) int {
	if match != nil && match.OpponentId != nil && *match.OpponentId == userId {
		return 2
	}
	return 1
}

func formatMatchID(matchId int64) string {
	return fmt.Sprintf("%d", matchId)
}
