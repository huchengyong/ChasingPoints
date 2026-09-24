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

type SendChallengeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 发起约球
func NewSendChallengeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendChallengeLogic {
	return &SendChallengeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *SendChallengeLogic) SendChallenge(req *types.SendChallengeReq) (resp *types.SendChallengeResp, err error) {
	resp = &types.SendChallengeResp{}
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		resp.Message = "获取用户信息失败"
		return resp, nil
	}
	if req.ToUserId <= 0 || req.ToUserId == userId {
		resp.Message = "请选择有效的平台对手"
		return resp, nil
	}

	mode, visibility, matchFormat, targetWins, snookerVersion, bestOfFrames, snookerTargetWins, startingActor, snookerFormat, optsErr :=
		validateChallengeMatchOptions(req.GameType, req.MatchMode, req.Visibility, req.MatchFormat, req.TargetWins,
			req.SnookerRulesVersion, req.BestOfFrames, req.SnookerTargetWins, req.StartingActor, req.SnookerFormat)
	if optsErr != nil {
		resp.Message = optsErr.Error()
		return resp, nil
	}
	// 事务外先做输入校验，无效请求不占用锁。
	if _, _, scheduleErr := ValidateChallengeSchedule(ChallengeScheduleInput{
		DayOffset:     req.DayOffset,
		ScheduledDate: req.ScheduledDate,
		StartHour:     req.StartHour,
		EndHour:       req.EndHour,
	}, time.Now()); scheduleErr != nil {
		resp.Message = scheduleErr.Error()
		return resp, nil
	}

	// 发送资格与接收方限制（事务外先读，事务内复核好友限制）。
	eligible, message, err := canSendChallengeTo(nil, l.svcCtx, userId, req.ToUserId)
	if err != nil {
		l.Logger.Errorf("检查约球资格失败: from=%d to=%d err=%v", userId, req.ToUserId, err)
		resp.Message = "检查约球资格失败"
		return resp, nil
	}
	if !eligible {
		resp.Message = message
		return resp, nil
	}

	var created *model.Challenge
	err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if err := l.svcCtx.UserModel.LockUsersForUpdate(tx, []int64{userId, req.ToUserId}); err != nil {
			return err
		}
		// 锁后再取当前时间：锁等待跨过失效时刻的旧约球不再占用发送资格。
		now := time.Now()
		// 锁后重新校验时段并计算失效：锁等待跨过区间结束的请求必须拒绝。
		scheduled, expiresAt, scheduleErr := ValidateChallengeSchedule(ChallengeScheduleInput{
			DayOffset:     req.DayOffset,
			ScheduledDate: req.ScheduledDate,
			StartHour:     req.StartHour,
			EndHour:       req.EndHour,
		}, now)
		if scheduleErr != nil {
			resp.Message = scheduleErr.Error()
			return nil
		}
		// 同账号最多一条有效已发出待回应邀请（并发发送也受限）。
		ownPending, err := l.svcCtx.ChallengeModel.FindActivePendingByFromUserWithTx(tx, userId, now)
		if err != nil {
			return err
		}
		if ownPending != nil {
			resp.PendingChallengeId = ownPending.Id
			resp.Message = "你发出的约球还在等待回应，请先处理"
			return nil
		}
		// 事务内按接收方最新设置与好友关系复核。
		eligible, message, err := canSendChallengeTo(tx, l.svcCtx, userId, req.ToUserId)
		if err != nil {
			return err
		}
		if !eligible {
			resp.Message = message
			return nil
		}
		// 已接受/等待中或由该约球开始的比赛尚未结束时，不能再发起。
		openJoined, err := l.svcCtx.ChallengeModel.FindOpenJoinedByUserWithTx(tx, userId, 0, now)
		if err != nil {
			return err
		}
		if openJoined != nil {
			resp.Message = "你已有未结束的约球，请先处理当前约球"
			return nil
		}

		challenge := &model.Challenge{
			FromUserId:          userId,
			ToUserId:            req.ToUserId,
			GameType:            req.GameType,
			ScheduledDate:       &scheduled,
			StartHour:           &req.StartHour,
			EndHour:             &req.EndHour,
			MatchMode:           mode,
			Visibility:          visibility,
			MatchFormat:         matchFormat,
			TargetWins:          targetWins,
			SnookerRulesVersion: snookerVersion,
			BestOfFrames:        bestOfFrames,
			SnookerFormat:       snookerFormat,
			SnookerTargetWins:   snookerTargetWins,
			StartingActor:       startingActor,
			Message:             req.Message,
			Status:              model.ChallengeStatusPending,
			ExpiresAt:           expiresAt,
		}
		if err := l.svcCtx.ChallengeModel.CreateWithTx(tx, challenge); err != nil {
			return err
		}
		created = challenge
		return nil
	})
	if err != nil {
		l.Logger.Errorf("发送约球失败: from=%d to=%d err=%v", userId, req.ToUserId, err)
		resp.Message = "发送失败"
		return resp, nil
	}
	if created == nil {
		return resp, nil
	}

	fromName := "球友"
	if fromUser, userErr := l.svcCtx.UserModel.FindById(userId); userErr == nil && fromUser != nil && fromUser.Nickname != "" {
		fromName = fromUser.Nickname
	}
	content := fromName + " 向你发起了约球"
	dispatchChallengeNotification(l.svcCtx, req.ToUserId, "你收到了一条约球", content, map[string]interface{}{
		"challenge_id": created.Id,
		"status":       model.ChallengeStatusPending,
	}, l.Logger)

	l.Logger.Infof("用户 %d 向 %d 发送约球 %d", userId, req.ToUserId, created.Id)
	return &types.SendChallengeResp{Success: true, ChallengeId: created.Id}, nil
}
