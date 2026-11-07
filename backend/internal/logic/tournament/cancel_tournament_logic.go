package tournament

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelTournamentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 取消赛事
func NewCancelTournamentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelTournamentLogic {
	return &CancelTournamentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *CancelTournamentLogic) CancelTournament(req *types.TournamentIdReq) (resp *types.CommonResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CommonResp{Success: false, Message: "获取用户信息失败"}, nil
	}
	if req.TournamentId <= 0 {
		return &types.CommonResp{Success: false, Message: "赛事参数无效"}, nil
	}

	// Only creator can cancel, and only status 0 (recruiting) can be cancelled
	tournament, err := l.svcCtx.TournamentModel.FindById(req.TournamentId)
	if err != nil {
		l.Logger.Errorf("查询赛事失败: id=%d err=%v", req.TournamentId, err)
		return &types.CommonResp{Success: false, Message: "操作失败"}, nil
	}
	if tournament == nil {
		return &types.CommonResp{Success: false, Message: "赛事不存在"}, nil
	}
	if tournament.CreatorId != userIdInt {
		return &types.CommonResp{Success: false, Message: "仅创建者可取消赛事"}, nil
	}
	if tournament.Status != 0 {
		return &types.CommonResp{Success: false, Message: "赛事已开始或已结束，无法取消"}, nil
	}

	// Update status to cancelled (3)
	updated, err := l.svcCtx.TournamentModel.UpdateStatusByCreator(req.TournamentId, userIdInt, 3)
	if err != nil {
		l.Logger.Errorf("取消赛事失败: id=%d err=%v", req.TournamentId, err)
		return &types.CommonResp{Success: false, Message: "操作失败"}, nil
	}
	if !updated {
		return &types.CommonResp{Success: false, Message: "取消失败"}, nil
	}

	return &types.CommonResp{Success: true, Message: "赛事已取消"}, nil
}
