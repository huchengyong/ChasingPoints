package match

import (
	"context"
	"errors"
	"fmt"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// finishTimeoutActionType 结束确认期满、由服务端按冻结比分完成结算时写入的系统动作。
const finishTimeoutActionType = "finish_timeout"

var errFinishRequestTimedOut = errors.New("确认期已满，对局已按冻结比分结束")

// expiredFinishCompletion 记录一次确认期满的结算，事务提交后用于广播与后续副作用。
type expiredFinishCompletion struct {
	match       *model.Match
	requestedBy int64
	deadline    time.Time
	settlement  finishMatchSettlement
}

// completeExpiredFinishRequestTx 在已加行锁的比赛上，若结束请求已到期，则按冻结比分完成结算：
// 先写系统动作 finish_timeout，再复用确认路径的结算，完成人记为发起方。未到期或已非进行中时返回 nil。
func completeExpiredFinishRequestTx(ctx context.Context, svcCtx *svc.ServiceContext, tx *gorm.DB, locked *model.Match, now time.Time) (*expiredFinishCompletion, error) {
	if locked == nil || locked.Status != 1 || !isFinishRequestExpired(locked, now) {
		return nil, nil
	}
	requestedBy := resolveFinishRequestedBy(locked)
	if requestedBy <= 0 {
		return nil, fmt.Errorf("结束请求缺少发起方: matchId=%d", locked.Id)
	}
	deadline := *locked.FinishRequestExpiresAt()
	requestRevision := locked.FinishRequestRevision
	clearFinishRequest(locked)
	timeoutRevision, err := svcCtx.MatchModel.BumpMatchRevisionWithTx(tx, locked)
	if err != nil {
		return nil, err
	}
	clientActionID := fmt.Sprintf("%s:%d:%d", finishTimeoutActionType, locked.Id, requestRevision)
	if err := svcCtx.MatchModel.CreateActionWithRevisionWithTx(tx, &model.MatchAction{
		MatchId:        locked.Id,
		RoundNo:        0,
		ActionType:     finishTimeoutActionType,
		Actor:          0,
		ClientActionId: stringPointer(clientActionID),
		BaseRevision:   requestRevision,
	}, timeoutRevision); err != nil {
		return nil, err
	}
	settlement, err := NewFinishMatchLogic(ctx, svcCtx).settleMatchWithCompletionSourceTx(tx, locked, requestedBy, &types.FinishMatchReq{
		MatchId:        locked.Id,
		ClientActionId: clientActionID + ":settle",
		BaseRevision:   timeoutRevision,
	}, model.CompletionSourceFinishTimeout)
	if err != nil {
		return nil, err
	}
	return &expiredFinishCompletion{match: locked, requestedBy: requestedBy, deadline: deadline, settlement: settlement}, nil
}

// finishExpiredRequestPostCommit 在确认期满结算提交后，执行与对方确认相同的广播、通知、成长与成就副作用。
func finishExpiredRequestPostCommit(ctx context.Context, svcCtx *svc.ServiceContext, completion *expiredFinishCompletion) {
	if completion == nil || completion.match == nil {
		return
	}
	match := completion.match
	broadcastFinishActionState(svcCtx, match, "match_finish_timeout")
	if _, err := NewFinishMatchLogic(ctx, svcCtx).finishMatchPostCommit(&types.FinishMatchReq{
		MatchId:        match.Id,
		ClientActionId: fmt.Sprintf("%s:%d", finishTimeoutActionType, match.Id),
	}, completion.requestedBy, match, completion.settlement.Result, completion.settlement.CompetitiveRevisions, completion.settlement.SeasonID); err != nil {
		logx.Errorf("确认期满结算后的后续处理失败: matchId=%d err=%v", match.Id, err)
	}
}

// completeExpiredFinishRequest 锁行复核后完成一场到期的结束请求；已被确认、撤回或已完成时不做任何事。
func completeExpiredFinishRequest(ctx context.Context, svcCtx *svc.ServiceContext, matchID int64, now time.Time) (*expiredFinishCompletion, error) {
	var completion *expiredFinishCompletion
	err := svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		locked, err := svcCtx.MatchModel.FindByIdForUpdateWithTx(tx, matchID)
		if err != nil || locked == nil {
			return err
		}
		completion, err = completeExpiredFinishRequestTx(ctx, svcCtx, tx, locked, now)
		return err
	})
	if err != nil {
		return nil, err
	}
	finishExpiredRequestPostCommit(ctx, svcCtx, completion)
	return completion, nil
}
