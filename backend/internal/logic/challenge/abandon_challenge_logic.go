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

type AbandonChallengeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 放弃本次约球：已接受未开局、另一方在等待（或无人等待）时，未进入方可放弃结束约球。
func NewAbandonChallengeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AbandonChallengeLogic {
	return &AbandonChallengeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *AbandonChallengeLogic) AbandonChallenge(req *types.HandleChallengeReq) (resp *types.ChallengeActionResp, err error) {
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
		if challenge.Status == model.ChallengeStatusStarted {
			matchId, err := l.svcCtx.MatchModel.FindIdByChallengeIdWithTx(tx, challenge.Id)
			if err != nil {
				return err
			}
			resp.MatchId = matchId
			resp.ChallengeId = challenge.Id
			resp.Message = "本次约球已正式开局，不能放弃"
			return nil
		}
		if challenge.Status != model.ChallengeStatusAccepted {
			resp.Message = "约球已处理或已失效"
			return nil
		}
		// worker 未运行时也按有效状态判断：已失效记录不能再改写成放弃。
		if !challenge.ExpiresAt.After(time.Now()) {
			resp.Message = "约球已失效"
			return nil
		}
		if challenge.WaitingUserId != nil && *challenge.WaitingUserId == userId {
			resp.Message = "你已进入等待，请使用退出等待或取消约球"
			return nil
		}
		ok, err := l.svcCtx.ChallengeModel.UpdateStatusWithTx(tx, challenge.Id, []int{model.ChallengeStatusAccepted}, model.ChallengeStatusAbandoned,
			map[string]interface{}{"close_reason": model.ChallengeCloseReasonAbandoned})
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
	})
	if err != nil {
		l.Logger.Errorf("放弃约球失败: id=%d err=%v", req.ChallengeId, err)
		// 事务已回滚：清除事务内写入的成功结果。
		resp.Success = false
		resp.MatchId = 0
		if resp.Message == "" {
			resp.Message = "操作失败"
		}
		return resp, nil
	}
	if resp.Success && notifyUserId > 0 {
		dispatchChallengeNotification(l.svcCtx, notifyUserId, "对方已放弃本次约球", "本次约球已结束，不会产生比赛", map[string]interface{}{
			"challenge_id": resp.ChallengeId,
			"status":       model.ChallengeStatusAbandoned,
		}, l.Logger)
	}
	return resp, nil
}
