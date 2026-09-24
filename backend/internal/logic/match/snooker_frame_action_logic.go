package match

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type SnookerFrameActionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 记录斯诺克局级动作
func NewSnookerFrameActionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SnookerFrameActionLogic {
	return &SnookerFrameActionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SnookerFrameActionLogic) SnookerFrameAction(req *types.SnookerFrameActionReq) (resp *types.SnookerActionResp, err error) {
	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return &types.SnookerActionResp{Success: false, Message: "用户未登录"}, nil
	}
	if req == nil {
		return &types.SnookerActionResp{Success: false, Message: "局动作参数不完整"}, nil
	}
	event := model.SnookerEvent{
		Version:     model.SnookerEventVersion,
		Kind:        model.SnookerEventKindFrameAction,
		Actor:       req.Actor,
		FrameAction: req.Action,
		Scope:       req.Scope,
		Winner:      req.Winner,
		Reason:      req.Reason,
	}
	return executeSnookerAction(l.ctx, l.svcCtx, userID, snookerActionWriteInput{
		MatchID:        req.MatchId,
		ClientActionID: req.ClientActionId,
		BaseRevision:   req.BaseRevision,
		Event:          event,
		ActionType:     model.MatchActionTypeSnookerFrameAction,
	})
}
