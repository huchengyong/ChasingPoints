package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminDeleteEventNewsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 删除赛事情报
func NewAdminDeleteEventNewsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminDeleteEventNewsLogic {
	return &AdminDeleteEventNewsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminDeleteEventNewsLogic) AdminDeleteEventNews(req *types.AdminEventNewsIdReq) (resp *types.AdminWriteResp, err error) {
	if _, err := utils.GetAdminIDFromCtx(l.ctx); err != nil {
		return &types.AdminWriteResp{
			Code:    401,
			Success: false,
			Message: "未登录或登录已过期",
		}, nil
	}

	if req == nil || req.EventId <= 0 {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "请求参数错误",
		}, nil
	}

	existing, err := l.svcCtx.EventNewsModel.FindById(req.EventId)
	if err != nil {
		l.Logger.Errorf("查询赛事情报失败: id=%d err=%v", req.EventId, err)
		return &types.AdminWriteResp{
			Code:    500,
			Success: false,
			Message: "删除赛事情报失败",
		}, nil
	}
	if existing == nil {
		return &types.AdminWriteResp{
			Code:    404,
			Success: false,
			Message: "赛事情报不存在",
		}, nil
	}

	if existing.TournamentId > 0 {
		if err := l.svcCtx.TournamentMatchModel.DeleteByTournament(existing.TournamentId); err != nil {
			l.Logger.Errorf("删除赛事比赛失败: eventId=%d tournamentId=%d err=%v", req.EventId, existing.TournamentId, err)
			return &types.AdminWriteResp{Code: 500, Success: false, Message: "删除赛事情报失败"}, nil
		}
		if err := l.svcCtx.TournamentModel.DeleteById(existing.TournamentId); err != nil {
			l.Logger.Errorf("删除赛事失败: eventId=%d tournamentId=%d err=%v", req.EventId, existing.TournamentId, err)
			return &types.AdminWriteResp{Code: 500, Success: false, Message: "删除赛事情报失败"}, nil
		}
	}

	if _, err := l.svcCtx.EventNewsModel.SoftDelete(req.EventId); err != nil {
		l.Logger.Errorf("删除赛事情报失败: id=%d err=%v", req.EventId, err)
		return &types.AdminWriteResp{
			Code:    500,
			Success: false,
			Message: "删除赛事情报失败",
		}, nil
	}

	return &types.AdminWriteResp{
		Code:    0,
		Success: true,
		Message: "删除成功",
	}, nil
}
