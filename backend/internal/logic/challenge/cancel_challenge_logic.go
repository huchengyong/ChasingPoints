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

type CancelChallengeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 取消约球：待回应时发起方可取消；已接受未开局时双方均可取消。
func NewCancelChallengeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelChallengeLogic {
	return &CancelChallengeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *CancelChallengeLogic) CancelChallenge(req *types.HandleChallengeReq) (resp *types.ChallengeActionResp, err error) {
	resp = &types.ChallengeActionResp{}
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		resp.Message = "获取用户信息失败"
		return resp, nil
	}
	if req.ChallengeId <= 0 {
		resp.Message = "约球不存在"
		return resp, nil
	}
	// 解析别名：旧通知可能指向归并记录。
	main, err := l.svcCtx.ChallengeModel.FindMainById(req.ChallengeId)
	if err != nil {
		l.Logger.Errorf("查询约球失败: id=%d err=%v", req.ChallengeId, err)
		resp.Message = "查询约球失败"
		return resp, nil
	}
	if main == nil {
		resp.Message = "约球不存在"
		return resp, nil
	}
	var notifyUserId int64
	err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		challenge, err := l.svcCtx.ChallengeModel.FindByIdForUpdateWithTx(tx, main.Id)
		if err != nil {
			return err
		}
		if challenge == nil {
			resp.Message = "约球不存在"
			return nil
		}
		if challenge.FromUserId != userId && challenge.ToUserId != userId {
			resp.Message = "只能处理自己的约球"
			return nil
		}
		// worker 未运行时也按有效状态判断：已失效记录不能再改写成取消。
		now := time.Now()
		if challenge.Status == model.ChallengeStatusStarted {
			matchId, err := l.svcCtx.MatchModel.FindIdByChallengeIdWithTx(tx, challenge.Id)
			if err != nil {
				return err
			}
			resp.MatchId = matchId
			resp.ChallengeId = challenge.Id
			resp.Message = "本次约球已正式开局，无法取消"
			return nil
		}
		if challenge.Status == model.ChallengeStatusPending {
			if challenge.FromUserId != userId {
				resp.Message = "只有发起人可以取消待回应邀请"
				return nil
			}
			if !challenge.ExpiresAt.After(now) {
				resp.Message = "约球已失效"
				return nil
			}
			ok, err := l.svcCtx.ChallengeModel.UpdateStatusWithTx(tx, challenge.Id, []int{model.ChallengeStatusPending}, model.ChallengeStatusCancelled,
				map[string]interface{}{"close_reason": model.ChallengeCloseReasonCancelled})
			if err != nil {
				return err
			}
			if !ok {
				resp.Message = "约球已处理或已失效"
				return nil
			}
			notifyUserId = challenge.ToUserId
			resp.Success = true
			resp.ChallengeId = challenge.Id
			return nil
		}
		if challenge.Status == model.ChallengeStatusAccepted {
			if !challenge.ExpiresAt.After(now) {
				resp.Message = "约球已失效"
				return nil
			}
			ok, err := l.svcCtx.ChallengeModel.UpdateStatusWithTx(tx, challenge.Id, []int{model.ChallengeStatusAccepted}, model.ChallengeStatusCancelled,
				map[string]interface{}{"close_reason": model.ChallengeCloseReasonCancelled})
			if err != nil {
				return err
			}
			if !ok {
				resp.Message = "约球已处理或已失效"
				return nil
			}
			notifyUserId = challenge.FromUserId
			if notifyUserId == userId {
				notifyUserId = challenge.ToUserId
			}
			resp.Success = true
			resp.ChallengeId = challenge.Id
			return nil
		}
		resp.Message = "约球已处理或已失效"
		return nil
	})
	if err != nil {
		l.Logger.Errorf("取消约球失败: id=%d err=%v", req.ChallengeId, err)
		// 事务已回滚：清除事务内写入的成功结果。
		resp.Success = false
		resp.MatchId = 0
		if resp.Message == "" {
			resp.Message = "操作失败"
		}
		return resp, nil
	}
	if resp.Success && notifyUserId > 0 {
		dispatchChallengeNotification(l.svcCtx, notifyUserId, "约球已取消", "对方取消了本次约球", map[string]interface{}{
			"challenge_id": resp.ChallengeId,
			"status":       model.ChallengeStatusCancelled,
		}, l.Logger)
	}
	return resp, nil
}
