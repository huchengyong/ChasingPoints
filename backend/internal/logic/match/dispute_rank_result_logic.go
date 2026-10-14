package match

import (
	"context"
	"fmt"
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

const rankResultDisputeActionType = "result_dispute"

type DisputeRankResultLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 对约球排位结果提出异议
func NewDisputeRankResultLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisputeRankResultLogic {
	return &DisputeRankResultLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DisputeRankResult 让被结束的一方在异议期内作废本场排位结算：对局与约球保持已完成，
// 比分与记录保留，仅本场不计排位积分。与到期结算 worker 在比赛行锁上串行，先提交者决定终态。
func (l *DisputeRankResultLogic) DisputeRankResult(req *types.RankResultDisputeReq) (resp *types.RankResultDisputeResp, err error) {
	if req == nil || req.MatchId <= 0 || req.ClientActionId == "" || l.svcCtx == nil || l.svcCtx.MatchModel == nil || l.svcCtx.DB == nil {
		return &types.RankResultDisputeResp{Success: false, Message: "请求参数无效"}, nil
	}
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return &types.RankResultDisputeResp{Success: false, Message: "获取用户信息失败"}, nil
	}

	var match *model.Match
	var message string
	replayed := false
	txErr := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		locked, findErr := l.svcCtx.MatchModel.FindByIdForUpdateWithTx(tx, req.MatchId)
		if findErr != nil {
			return findErr
		}
		if locked == nil {
			message = "对局不存在"
			return nil
		}
		match = locked
		if !isMatchPlayer(locked, userId) {
			message = "你不是本场对局参与者"
			return nil
		}
		existing, findErr := l.svcCtx.MatchModel.FindActionByClientActionIDWithTx(tx, locked.Id, req.ClientActionId)
		if findErr != nil {
			return findErr
		}
		if existing != nil {
			if existing.ActionType == rankResultDisputeActionType && existing.Actor == resolveFinishActionActor(locked, userId) &&
				locked.RankSettlementStatus == model.RankSettlementVoided {
				replayed = true
				return nil
			}
			message = "该请求标识已被其他操作使用"
			return nil
		}
		if message = rankDisputeRejection(locked, userId, time.Now()); message != "" {
			return nil
		}

		locked.RankSettlementStatus = model.RankSettlementVoided
		revision, bumpErr := l.svcCtx.MatchModel.BumpMatchRevisionWithTx(tx, locked)
		if bumpErr != nil {
			return bumpErr
		}
		return l.svcCtx.MatchModel.CreateActionWithRevisionWithTx(tx, &model.MatchAction{
			MatchId:        locked.Id,
			RoundNo:        0,
			ActionType:     rankResultDisputeActionType,
			Actor:          resolveFinishActionActor(locked, userId),
			ClientActionId: stringPointer(req.ClientActionId),
			BaseRevision:   revision - 1,
		}, revision)
	})
	if txErr != nil {
		l.Logger.Errorf("异议事务失败: matchId=%d userId=%d err=%v", req.MatchId, userId, txErr)
		message = "当前无法提出异议，请稍后重试"
	}
	if message != "" {
		return l.buildResponse(userId, req, l.reloadForResponse(req.MatchId, match), false, message), nil
	}

	if !replayed {
		finalizer := NewFinishMatchLogic(l.ctx, l.svcCtx)
		finalizer.finalizeRankVoided(match)
		l.notifyFinisher(userId, match)
	}
	return l.buildResponse(userId, req, match, true, "已提出异议，本场不计排位积分"), nil
}

// rankDisputeRejection 返回不能提出异议的原因；返回空串表示可以提出。
func rankDisputeRejection(match *model.Match, userId int64, now time.Time) string {
	switch {
	case match.Status != 2:
		return "对局尚未结束"
	case match.ChallengeId == nil || *match.ChallengeId <= 0:
		return "本场对局不支持异议"
	case match.RankSettlementStatus == model.RankSettlementVoided:
		return "本场已提出异议"
	case match.RankSettlementStatus == model.RankSettlementSettled:
		return "异议期已结束，排位结果已生效"
	case match.RankSettlementStatus != model.RankSettlementPending:
		return "本场无需异议"
	case match.CompletedByUserId != nil && *match.CompletedByUserId == userId:
		return "你是结束方，无法对自己的结果提出异议"
	case match.RankSettlementDueAt == nil || !now.Before(*match.RankSettlementDueAt):
		return "异议期已结束，排位结果已生效"
	case !match.CanDisputeRankResult(userId, now):
		return "当前无法提出异议"
	}
	return ""
}

func (l *DisputeRankResultLogic) reloadForResponse(matchID int64, fallback *model.Match) *model.Match {
	if fresh, err := l.svcCtx.MatchModel.FindById(matchID); err == nil && fresh != nil {
		return fresh
	}
	return fallback
}

func (l *DisputeRankResultLogic) buildResponse(userId int64, req *types.RankResultDisputeReq, match *model.Match, success bool, message string) *types.RankResultDisputeResp {
	resp := &types.RankResultDisputeResp{Success: success, Message: message, ClientActionId: req.ClientActionId}
	if match == nil {
		return resp
	}
	resp.RankSettlementStatus = resolveRankSettlementStatus(match)
	resp.RankSettlementDueAt = match.FormatRankSettlementDueAt()
	if view, err := loadMatchWriteState(l.svcCtx, userId, match); err == nil {
		resp.ServerRevision = view.Snapshot.ServerRevision
		resp.Snapshot = view.Snapshot
	}
	return resp
}

// notifyFinisher 提交后通知结束方对方已提出异议；通知失败只记录，不影响异议结果。
func (l *DisputeRankResultLogic) notifyFinisher(disputerID int64, match *model.Match) {
	if match == nil || match.CompletedByUserId == nil || *match.CompletedByUserId <= 0 || l.svcCtx.UserModel == nil {
		return
	}
	name := "对方"
	if user, err := l.svcCtx.UserModel.FindById(disputerID); err == nil && user != nil && user.Nickname != "" {
		name = user.Nickname
	}
	content := fmt.Sprintf("%s对本场结果提出异议，本场不计排位积分", name)
	if err := logicx.DispatchNotification(l.svcCtx, logicx.NotificationDispatchInput{
		UserId:      *match.CompletedByUserId,
		Type:        "match_result",
		DedupeKey:   fmt.Sprintf("match:%d:dispute", match.Id),
		Title:       "对局结果有异议",
		Content:     content,
		Data:        buildNotificationPayload("/subPages/match/matchResult", match.Id, 0),
		PushTitle:   "对局结果有异议",
		PushContent: content,
		PushData: map[string]interface{}{
			"url": fmt.Sprintf("/subPages/match/matchResult?match_id=%d", match.Id),
		},
		WSCategory: "match_result",
	}); err != nil {
		l.Logger.Errorf("分发异议通知失败: matchId=%d userId=%d err=%v", match.Id, *match.CompletedByUserId, err)
	}
}
