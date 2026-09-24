package challenge

import (
	"context"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type AcceptChallengeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 接受约球
func NewAcceptChallengeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AcceptChallengeLogic {
	return &AcceptChallengeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *AcceptChallengeLogic) AcceptChallenge(req *types.AcceptChallengeReq) (resp *types.AcceptChallengeResp, err error) {
	resp = &types.AcceptChallengeResp{}
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		resp.Message = "获取用户信息失败"
		return resp, nil
	}
	if req.ChallengeId <= 0 {
		resp.Message = "约球不存在"
		return resp, nil
	}

	// 事务外先解析别名，拿到主记录 ID 供锁使用。
	main, err := l.svcCtx.ChallengeModel.FindMainById(req.ChallengeId)
	if err != nil {
		l.Logger.Errorf("查询约球失败: id=%d err=%v", req.ChallengeId, err)
		resp.Message = "查询约球失败"
		return resp, nil
	}
	if main == nil {
		resp.Message = "约球不存在或已处理"
		return resp, nil
	}
	resp.ChallengeId = main.Id

	// 仅用无锁查询确定可能涉及的第三位用户；事务仍先锁全部用户，再锁约球行。
	ownHint, err := l.svcCtx.ChallengeModel.FindActivePendingToOtherWithTx(nil, userId, main.FromUserId, time.Now())
	if err != nil {
		return &types.AcceptChallengeResp{Message: "查询已发出约球失败"}, nil
	}
	lockIds := []int64{userId, main.FromUserId}
	if ownHint != nil {
		lockIds = append(lockIds, ownHint.ToUserId)
	}
	var switchedPending *model.Challenge
	var mutualMerged bool
	err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if err := l.svcCtx.UserModel.LockUsersForUpdate(tx, buildLockUserIds(lockIds...)); err != nil {
			return err
		}
		locked, err := l.svcCtx.ChallengeModel.FindByIdForUpdateWithTx(tx, main.Id)
		if err != nil {
			return err
		}
		if locked == nil {
			resp.Message = "约球不存在或已处理"
			return nil
		}
		// 事务内重查别名（可能刚被归并）。
		if locked.MergedIntoId != nil && *locked.MergedIntoId > 0 {
			locked, err = l.svcCtx.ChallengeModel.FindByIdForUpdateWithTx(tx, *locked.MergedIntoId)
			if err != nil {
				return err
			}
			if locked == nil {
				resp.Message = "约球不存在或已处理"
				return nil
			}
			resp.ChallengeId = locked.Id
		}
		// 主记录也已锁定；锁等待跨过 07:00 的请求不能接受。
		now := time.Now()

		if locked.FromUserId != userId && locked.ToUserId != userId {
			resp.Message = "只能处理自己的约球"
			return nil
		}
		// 重复点击或点到已接受/已合并记录：返回现有约球，不新建；已失效的已接受记录不得谎报成功。
		if locked.Status == model.ChallengeStatusAccepted || locked.Status == model.ChallengeStatusStarted {
			if locked.Status == model.ChallengeStatusAccepted && !locked.ExpiresAt.After(now) {
				resp.Message = "约球已失效"
				return nil
			}
			resp.Success = true
			if locked.Status == model.ChallengeStatusStarted {
				matchId, err := l.svcCtx.MatchModel.FindIdByChallengeIdWithTx(tx, locked.Id)
				if err != nil {
					return err
				}
				resp.MatchId = matchId
			}
			return nil
		}
		if locked.Status != model.ChallengeStatusPending {
			resp.Message = "约球已处理或已失效"
			return nil
		}
		if locked.ToUserId != userId {
			resp.Message = "只有被约球人可以接受"
			return nil
		}
		if !locked.ExpiresAt.After(now) {
			resp.Message = "约球已失效"
			return nil
		}

		// 已有未结束约球：不能再接受（接收人与发送人都受唯一占用限制）。
		openJoined, err := l.svcCtx.ChallengeModel.FindOpenJoinedByUserWithTx(tx, userId, locked.Id, now)
		if err != nil {
			return err
		}
		if openJoined != nil {
			resp.Message = "你已有未结束的约球，请先处理当前约球"
			return nil
		}
		senderJoined, err := l.svcCtx.ChallengeModel.FindOpenJoinedByUserWithTx(tx, locked.FromUserId, locked.Id, now)
		if err != nil {
			return err
		}
		if senderJoined != nil {
			resp.Message = "对方已有未结束的约球，请稍后再试"
			return nil
		}

		// 互邀合并：同一人、同球种、同比赛类型；时间不同以本次被接受的邀请为准。
		mutual, err := l.svcCtx.ChallengeModel.FindMutualPendingWithTx(tx, userId, locked.FromUserId, locked.GameType, locked.MatchMode, now)
		if err != nil {
			return err
		}
		if mutual != nil && mutual.Id != locked.Id {
			ok, err := l.svcCtx.ChallengeModel.UpdateStatusWithTx(tx, mutual.Id, []int{model.ChallengeStatusPending}, model.ChallengeStatusMerged,
				map[string]interface{}{"merged_into_id": locked.Id, "close_reason": model.ChallengeCloseReasonMerged})
			if err != nil {
				return err
			}
			if ok {
				mutualMerged = true
			}
		} else if mutual == nil {
			// 同人不同类型：原状返回，不合并不改类型。
			hasAny, err := l.svcCtx.ChallengeModel.HasAnyMutualPendingWithTx(tx, userId, locked.FromUserId, now)
			if err != nil {
				return err
			}
			if hasAny {
				resp.TypeConflict = true
				resp.Message = "你们发起的约球类型不同，不能直接匹配"
				return nil
			}
		}

		// 换人接受：自己发给其他人的待回应邀请需要先关闭。
		ownPending, err := l.svcCtx.ChallengeModel.FindActivePendingToOtherWithTx(tx, userId, locked.FromUserId, now)
		if err != nil {
			return err
		}
		if ownPending != nil {
			if ownHint == nil || ownHint.ToUserId != ownPending.ToUserId {
				resp.Message = "已发出的约球已变化，请重新确认"
				return nil
			}
			if !req.ConfirmCloseOwnPending {
				resp.NeedConfirmCloseOwn = true
				resp.PendingChallengeId = ownPending.Id
				resp.Message = "接受后会自动关闭已发出的邀请"
				return nil
			}
			ok, err := l.svcCtx.ChallengeModel.UpdateStatusWithTx(tx, ownPending.Id, []int{model.ChallengeStatusPending}, model.ChallengeStatusCancelled,
				map[string]interface{}{"close_reason": model.ChallengeCloseReasonSwitched})
			if err != nil {
				return err
			}
			if ok {
				copied := *ownPending
				switchedPending = &copied
			}
		}

		ok, err := l.svcCtx.ChallengeModel.UpdateStatusWithTx(tx, locked.Id, []int{model.ChallengeStatusPending}, model.ChallengeStatusAccepted, nil)
		if err != nil {
			return err
		}
		if !ok {
			// 并发下已被接受/处理：重查返回权威状态。
			fresh, err := l.svcCtx.ChallengeModel.FindByIdForUpdateWithTx(tx, locked.Id)
			if err != nil {
				return err
			}
			if fresh != nil && (fresh.Status == model.ChallengeStatusAccepted || fresh.Status == model.ChallengeStatusStarted) {
				if fresh.Status == model.ChallengeStatusAccepted && !fresh.ExpiresAt.After(now) {
					resp.Message = "约球已失效"
					return nil
				}
				resp.Success = true
				if fresh.Status == model.ChallengeStatusStarted {
					matchId, err := l.svcCtx.MatchModel.FindIdByChallengeIdWithTx(tx, fresh.Id)
					if err != nil {
						return err
					}
					resp.MatchId = matchId
				}
				return nil
			}
			resp.Message = "约球已处理或已失效"
			return nil
		}
		resp.Success = true
		return nil
	})
	if err != nil {
		l.Logger.Errorf("接受约球失败: id=%d userId=%d err=%v", req.ChallengeId, userId, err)
		// 事务已回滚：清除事务内写入的成功与确认标记。
		resp.Success = false
		resp.MatchId = 0
		resp.NeedConfirmCloseOwn = false
		resp.PendingChallengeId = 0
		if resp.Message == "" {
			resp.Message = "操作失败"
		}
		return resp, nil
	}

	if resp.Success {
		if mutualMerged {
			l.Logger.Infof("互邀合并: main=%d by=%d", resp.ChallengeId, userId)
		}
		if switchedPending != nil {
			dispatchChallengeNotification(l.svcCtx, switchedPending.ToUserId, "约球已取消", "对方已取消本次邀请", map[string]interface{}{
				"challenge_id": switchedPending.Id,
				"status":       model.ChallengeStatusCancelled,
			}, l.Logger)
		}
		accepted, findErr := l.svcCtx.ChallengeModel.FindById(resp.ChallengeId)
		if findErr == nil && accepted != nil && accepted.ToUserId == userId {
			fromName := "球友"
			if fromUser, userErr := l.svcCtx.UserModel.FindById(userId); userErr == nil && fromUser != nil && fromUser.Nickname != "" {
				fromName = fromUser.Nickname
			}
			dispatchChallengeNotification(l.svcCtx, accepted.FromUserId, "约球已接受", fromName+" 接受了你的约球", map[string]interface{}{
				"challenge_id": accepted.Id,
				"status":       model.ChallengeStatusAccepted,
			}, l.Logger)
		}
	}
	return resp, nil
}
