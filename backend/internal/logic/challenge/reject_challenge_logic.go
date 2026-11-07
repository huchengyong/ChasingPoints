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

type RejectChallengeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 拒绝约球
func NewRejectChallengeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RejectChallengeLogic {
	return &RejectChallengeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *RejectChallengeLogic) RejectChallenge(req *types.HandleChallengeReq) (resp *types.ChallengeActionResp, err error) {
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
	var senderId int64
	err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		challenge, err := l.svcCtx.ChallengeModel.FindByIdForUpdateWithTx(tx, req.ChallengeId)
		if err != nil {
			return err
		}
		// 锁后再取当前时间：锁等待跨过失效时刻的请求不能拒绝成功。
		now := time.Now()
		if challenge == nil || challenge.ToUserId != userId {
			resp.Message = "约球不存在"
			return nil
		}
		if challenge.Status != model.ChallengeStatusPending {
			resp.Message = "约球已处理或已失效"
			return nil
		}
		if !challenge.ExpiresAt.After(now) {
			resp.Message = "约球已失效"
			return nil
		}
		ok, err := l.svcCtx.ChallengeModel.UpdateStatusWithTx(tx, challenge.Id, []int{model.ChallengeStatusPending}, model.ChallengeStatusRejected,
			map[string]interface{}{"close_reason": model.ChallengeCloseReasonRejected})
		if err != nil {
			return err
		}
		if !ok {
			resp.Message = "约球已处理或已失效"
			return nil
		}
		senderId = challenge.FromUserId
		resp.Success = true
		resp.ChallengeId = challenge.Id
		return nil
	})
	if err != nil {
		l.Logger.Errorf("拒绝约球失败: id=%d err=%v", req.ChallengeId, err)
		// 事务已回滚：清除事务内写入的成功结果。
		resp.Success = false
		if resp.Message == "" {
			resp.Message = "操作失败"
		}
		return resp, nil
	}
	if resp.Success {
		dispatchChallengeNotification(l.svcCtx, senderId, "约球已拒绝", "对方这次不了", map[string]interface{}{
			"challenge_id": resp.ChallengeId,
			"status":       model.ChallengeStatusRejected,
		}, l.Logger)
	}
	return resp, nil
}
