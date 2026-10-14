package match

import (
	"context"
	"fmt"
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/gorm"
)

// settleOptions 描述一次结束动作的来源，用于决定排位结算是否延后。
type settleOptions struct {
	// manualFinish 表示由参赛方通过 /api/match/finish 手动结束（含滑动结束）；
	// 赛制达标自动结束、裁判结束、认输等路径不是手动结束。
	manualFinish bool
}

type rankSettlementPlan struct {
	status string
	dueAt  *time.Time
}

// planRankSettlement 决定本次结束的排位结算方式：
//   - 约球排位的胜负结果：参赛方手动结束 → pending，异议期后生效；其余路径 → 立即结算（settled）。
//   - 平局、练习赛与非约球比赛 → none，沿用既有流程。
func planRankSettlement(match *model.Match, result int, finisherUserId int64, opts settleOptions, now time.Time) rankSettlementPlan {
	if match == nil || match.ChallengeId == nil || *match.ChallengeId <= 0 ||
		model.NormalizeMatchMode(match.MatchMode) != model.MatchModeRanked || (result != 1 && result != 2) {
		return rankSettlementPlan{status: model.RankSettlementNone}
	}
	if !opts.manualFinish || !isMatchPlayer(match, finisherUserId) {
		return rankSettlementPlan{status: model.RankSettlementSettled}
	}
	due := now.Add(model.RankDisputeWindow)
	return rankSettlementPlan{status: model.RankSettlementPending, dueAt: &due}
}

func isMatchPlayer(match *model.Match, userId int64) bool {
	if match == nil || userId <= 0 {
		return false
	}
	return match.UserId == userId || (match.OpponentId != nil && *match.OpponentId == userId)
}

// resolveRankSettlementStatus 返回对外的排位结算状态，旧数据与空值按 none 处理。
func resolveRankSettlementStatus(match *model.Match) string {
	if match == nil || match.RankSettlementStatus == "" {
		return model.RankSettlementNone
	}
	return match.RankSettlementStatus
}

// rankAffecting 表示该场结果已经（或应当立即）影响排位口径数据：排位且有胜负，并且排位结算未被延后或作废。
func rankAffecting(match *model.Match, result int) bool {
	if match == nil || result == 3 || model.NormalizeMatchMode(match.MatchMode) != model.MatchModeRanked {
		return false
	}
	return !match.RankSettlementDeferred()
}

// rankSettlementOutcomeKind 描述一次到期处理的结果。
type rankSettlementOutcomeKind int

const (
	rankSettlementSkipped             rankSettlementOutcomeKind = iota // 未到期、已被异议或已处理
	rankSettlementApplied                                              // 排位结算已生效
	rankSettlementVoidedByMissingUser                                  // 参赛账号已不存在，转为作废
)

// settleDueRankMatch 把一场已到生效时刻的待生效排位结算落库，返回提交后的比赛。
// 全程在比赛行锁内复核 pending 与到期时刻，多实例并发或重复调用至多结算一次。
func settleDueRankMatch(ctx context.Context, svcCtx *svc.ServiceContext, matchID int64, now time.Time) (*model.Match, finishMatchSettlement, rankSettlementOutcomeKind, error) {
	logic := NewFinishMatchLogic(ctx, svcCtx)
	var settled *model.Match
	var settlement finishMatchSettlement
	kind := rankSettlementSkipped
	err := svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		locked, err := svcCtx.MatchModel.FindByIdForUpdateWithTx(tx, matchID)
		if err != nil {
			return err
		}
		if locked == nil || locked.Status != 2 || locked.Result == nil || locked.RankSettlementStatus != model.RankSettlementPending ||
			locked.RankSettlementDueAt == nil || locked.RankSettlementDueAt.After(now) {
			return nil
		}
		result := *locked.Result
		effectiveAt := *locked.RankSettlementDueAt

		missing, err := rankSettlementPlayerMissing(tx, svcCtx, locked)
		if err != nil {
			return err
		}
		if missing {
			// 账号已注销的比赛不再写其段位：按作废处理，保留赛果记录。
			locked.RankSettlementStatus = model.RankSettlementVoided
			if _, err := recordRankSettlementTransitionWithTx(tx, svcCtx, locked, rankVoidedActionType); err != nil {
				return err
			}
			settled, kind = locked, rankSettlementVoidedByMissingUser
			return nil
		}

		facts, err := logic.loadRankSettlementFactsTx(tx, locked, result)
		if err != nil {
			return err
		}
		outcome, err := logic.applyRankSettlementTx(tx, locked, effectiveAt, facts)
		if err != nil {
			return err
		}
		locked.RankSettlementStatus = model.RankSettlementSettled
		revision, err := recordRankSettlementTransitionWithTx(tx, svcCtx, locked, rankSettledActionType)
		if err != nil {
			return err
		}
		projection, err := applyDeferredRankStatsWithTx(tx, svcCtx, logicx.CompetitiveProjectionInput{
			Match:             locked,
			Player1Before:     outcome.Player1Before,
			Player1After:      outcome.Player1After,
			Player2Before:     outcome.Player2Before,
			Player2After:      outcome.Player2After,
			SeasonEffectiveAt: &effectiveAt,
		})
		if err != nil {
			return err
		}
		settled, kind = locked, rankSettlementApplied
		settlement = finishMatchSettlement{
			Result:               result,
			ServerRevision:       revision,
			CompetitiveRevisions: projection.CompetitiveRevisions,
			SeasonID:             projection.SeasonID,
		}
		return nil
	})
	return settled, settlement, kind, err
}

const (
	rankSettledActionType = "rank_settled"
	rankVoidedActionType  = "rank_voided"
)

// recordRankSettlementTransitionWithTx 在比赛行锁内持久化新的排位结算状态，并推进 revision、写入系统审计动作，
// 让客户端能通过 revision 感知变化，也避免持有旧副本的写入把状态覆盖回去。
func recordRankSettlementTransitionWithTx(tx *gorm.DB, svcCtx *svc.ServiceContext, match *model.Match, actionType string) (int64, error) {
	revision, err := svcCtx.MatchModel.BumpMatchRevisionWithTx(tx, match)
	if err != nil {
		return 0, err
	}
	if err := svcCtx.MatchModel.CreateActionWithRevisionWithTx(tx, &model.MatchAction{
		MatchId:      match.Id,
		RoundNo:      0,
		ActionType:   actionType,
		Actor:        0,
		BaseRevision: revision - 1,
	}, revision); err != nil {
		return 0, err
	}
	return revision, nil
}

func rankSettlementPlayerMissing(tx *gorm.DB, svcCtx *svc.ServiceContext, match *model.Match) (bool, error) {
	ids := []int64{match.UserId}
	if match.OpponentId != nil && *match.OpponentId > 0 {
		ids = append(ids, *match.OpponentId)
	}
	for _, id := range ids {
		user, err := svcCtx.UserModel.FindByIdWithTx(tx, id)
		if err != nil {
			return false, err
		}
		if user == nil {
			return true, nil
		}
	}
	return false, nil
}

func applyDeferredRankStatsWithTx(tx *gorm.DB, svcCtx *svc.ServiceContext, input logicx.CompetitiveProjectionInput) (logicx.CompetitiveProjectionResult, error) {
	if svcCtx == nil || svcCtx.CompetitiveReadModel == nil {
		return logicx.CompetitiveProjectionResult{
			AppliedUsers:         map[int64]bool{},
			CompetitiveRevisions: map[int64]int64{},
		}, nil
	}
	return logicx.NewCompetitiveProjector(svcCtx).ApplyDeferredRankStatsWithTx(tx, input)
}

// finalizeRankSettled 在延后结算提交后补做原本随结束事务提交后执行的排位副作用：
// 信誉、排位/统计失效与广播、会员成长和成就进度。
func (l *FinishMatchLogic) finalizeRankSettled(match *model.Match, settlement finishMatchSettlement) {
	if match == nil || match.Result == nil {
		return
	}
	result := *match.Result
	l.applyMatchReputation(match)
	broadcastRankInfoUpdated(match, result)
	broadcastUserDataUpdated(match, result, settlement.CompetitiveRevisions)
	l.invalidateCompetitiveSharedReads(match, result, settlement.SeasonID)
	l.awardMemberGrowthForMatch(match.Id, match.UserId, match.OpponentId)
	l.syncAchievementProgressForCompletedMatch(match)
}

// finalizeRankVoided 在排位结算作废后按非排位处理：不产生排位成就事件，只刷新双方历史与活动列表。
func (l *FinishMatchLogic) finalizeRankVoided(match *model.Match) {
	if match == nil || match.Result == nil {
		return
	}
	l.syncAchievementProgressForCompletedMatch(match)
	broadcastUserDataUpdated(match, *match.Result, nil)
}

// dispatchChallengeFinishNotices 在约球比赛由参赛方结束后，通知另一名参赛方“X结束了对局 · 比分”；
// 排位待生效时附带异议期提示。返回已收到该提醒的用户，调用方据此跳过重复的结果通知。
func (l *FinishMatchLogic) dispatchChallengeFinishNotices(match *model.Match) map[int64]bool {
	sent := map[int64]bool{}
	if match == nil || match.ChallengeId == nil || *match.ChallengeId <= 0 || match.CompletedByUserId == nil ||
		!isMatchPlayer(match, *match.CompletedByUserId) || match.OpponentId == nil || *match.OpponentId <= 0 || l.svcCtx.UserModel == nil {
		return sent
	}
	finisherID := *match.CompletedByUserId
	recipientID := match.UserId
	recipientScore, otherScore := match.MyScore, match.OpponentScore
	if finisherID == match.UserId {
		recipientID = *match.OpponentId
		recipientScore, otherScore = match.OpponentScore, match.MyScore
	}
	name := "对方"
	if user, err := l.svcCtx.UserModel.FindById(finisherID); err == nil && user != nil && user.Nickname != "" {
		name = user.Nickname
	}
	content := fmt.Sprintf("%s结束了对局 · %d:%d", name, recipientScore, otherScore)
	if match.RankSettlementStatus == model.RankSettlementPending {
		content += fmt.Sprintf("，%d分钟内可提出异议", int(model.RankDisputeWindow.Minutes()))
	}
	if err := logicx.DispatchNotification(l.svcCtx, logicx.NotificationDispatchInput{
		UserId:      recipientID,
		Type:        "match_result",
		DedupeKey:   fmt.Sprintf("match:%d", match.Id),
		Title:       "对局已结束",
		Content:     content,
		Data:        buildNotificationPayload("/subPages/match/matchResult", match.Id, 0),
		PushTitle:   "对局已结束",
		PushContent: content,
		PushData: map[string]interface{}{
			"url": fmt.Sprintf("/subPages/match/matchResult?match_id=%d", match.Id),
		},
		WSCategory: "match_result",
	}); err != nil {
		l.Logger.Errorf("分发约球结束提醒失败: matchId=%d userId=%d err=%v", match.Id, recipientID, err)
		return sent
	}
	sent[recipientID] = true
	return sent
}
