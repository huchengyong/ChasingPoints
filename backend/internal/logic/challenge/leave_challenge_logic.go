package challenge

import (
	"context"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type LeaveChallengeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 退出等待，保留约球
func NewLeaveChallengeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LeaveChallengeLogic {
	return &LeaveChallengeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *LeaveChallengeLogic) LeaveChallenge(req *types.HandleChallengeReq) (resp *types.ChallengeActionResp, err error) {
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
	resp.ChallengeId = main.Id
	var otherUser int64
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
			resp.MatchId, err = l.svcCtx.MatchModel.FindIdByChallengeIdWithTx(tx, challenge.Id)
			if err != nil {
				return err
			}
			resp.Message = "比赛已开始，请继续比赛"
			return nil
		}
		// 幂等：本人已退出等待时可重试；终态不能误报退出成功。
		if challenge.Status != model.ChallengeStatusAccepted {
			resp.Message = "约球已结束"
			return nil
		}
		if challenge.WaitingUserId == nil || *challenge.WaitingUserId != userId {
			resp.Success = true
			return nil
		}
		ok, err := l.svcCtx.ChallengeModel.ClearWaitingWithTx(tx, challenge.Id, userId)
		if err != nil {
			return err
		}
		resp.Success = ok
		if !ok {
			resp.Message = "约球状态已变化，请稍后重试"
		} else {
			otherUser = challenge.FromUserId
			if otherUser == userId {
				otherUser = challenge.ToUserId
			}
		}
		return nil
	})
	if err != nil {
		l.Logger.Errorf("退出等待失败: id=%d err=%v", req.ChallengeId, err)
		// 事务已回滚：清除事务内写入的成功结果。
		resp.Success = false
		resp.MatchId = 0
		if resp.Message == "" {
			resp.Message = "操作失败"
		}
		return resp, nil
	}
	if resp.Success && otherUser > 0 {
		logicx.SendUserDataUpdated(otherUser, logicx.UserDataUpdatedEvent{Scopes: []string{"challenge"}})
	}
	return resp, nil
}
