package tournament

import (
	"context"

	"chasing_points/internal/config"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type CheckinTournamentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 签到赛事
func NewCheckinTournamentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckinTournamentLogic {
	return &CheckinTournamentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *CheckinTournamentLogic) CheckinTournament(req *types.TournamentIdReq) (resp *types.CommonResp, err error) {
	if !l.svcCtx.Config.UserTournamentEnabled() {
		return &types.CommonResp{Success: false, Message: config.DisabledFeatureMessage("tournament_user_action")}, nil
	}

	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CommonResp{Success: false, Message: "获取用户信息失败"}, nil
	}
	if req.TournamentId <= 0 {
		return &types.CommonResp{Success: false, Message: "赛事参数无效"}, nil
	}

	tournament, err := l.svcCtx.TournamentModel.FindById(req.TournamentId)
	if err != nil {
		l.Logger.Errorf("查询赛事失败: tournamentId=%d err=%v", req.TournamentId, err)
		return &types.CommonResp{Success: false, Message: "操作失败"}, nil
	}
	if tournament == nil {
		return &types.CommonResp{Success: false, Message: "赛事不存在"}, nil
	}
	if tournament.Status != 0 {
		return &types.CommonResp{Success: false, Message: "当前不可签到"}, nil
	}

	// Verify participant exists
	participant, err := l.svcCtx.TournamentParticipantModel.FindByTournamentAndUser(req.TournamentId, userIdInt)
	if err != nil {
		l.Logger.Errorf("查询参赛者失败: tournamentId=%d userId=%d err=%v", req.TournamentId, userIdInt, err)
		return &types.CommonResp{Success: false, Message: "操作失败"}, nil
	}
	if participant == nil {
		return &types.CommonResp{Success: false, Message: "未报名该赛事"}, nil
	}
	if participant.Status != 0 {
		return &types.CommonResp{Success: false, Message: "已签到或状态不允许"}, nil
	}

	// Update status to checked in (1)
	updated, err := l.svcCtx.TournamentParticipantModel.UpdateStatusByTournamentAndUser(req.TournamentId, userIdInt, 1)
	if err != nil {
		l.Logger.Errorf("签到失败: tournamentId=%d userId=%d err=%v", req.TournamentId, userIdInt, err)
		return &types.CommonResp{Success: false, Message: "签到失败"}, nil
	}
	if !updated {
		return &types.CommonResp{Success: false, Message: "签到失败"}, nil
	}

	return &types.CommonResp{Success: true, Message: "签到成功"}, nil
}
