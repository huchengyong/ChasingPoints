package match

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type SnookerStrokeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 记录斯诺克击球结果
func NewSnookerStrokeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SnookerStrokeLogic {
	return &SnookerStrokeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SnookerStrokeLogic) SnookerStroke(req *types.SnookerStrokeReq) (resp *types.SnookerActionResp, err error) {
	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return &types.SnookerActionResp{Success: false, Message: "用户未登录"}, nil
	}
	if req == nil {
		return &types.SnookerActionResp{Success: false, Message: "击球参数不完整"}, nil
	}
	event := model.SnookerEvent{
		Version:              model.SnookerEventVersion,
		Kind:                 model.SnookerEventKindStroke,
		Actor:                req.Actor,
		Outcome:              req.Outcome,
		BallOnValue:          req.BallOnValue,
		PottedReds:           req.PottedReds,
		BallOnPotted:         req.BallOnPotted,
		FreeBallValue:        req.FreeBallValue,
		FreeBallPotted:       req.FreeBallPotted,
		Penalty:              req.Penalty,
		RedsRemoved:          req.RedsRemoved,
		FoulAndMiss:          req.FoulAndMiss,
		MissSequenceEligible: req.MissSequenceEligible,
		FoulResolution:       req.FoulResolution,
		FreeBallAwarded:      req.FreeBallAwarded,
		CueBallInHand:        req.CueBallInHand,
	}
	return executeSnookerAction(l.ctx, l.svcCtx, userID, snookerActionWriteInput{
		MatchID:        req.MatchId,
		ClientActionID: req.ClientActionId,
		BaseRevision:   req.BaseRevision,
		Event:          event,
		ActionType:     model.MatchActionTypeSnookerStroke,
	})
}
