package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminUpdateEventNewsMatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新赛事比赛
func NewAdminUpdateEventNewsMatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminUpdateEventNewsMatchLogic {
	return &AdminUpdateEventNewsMatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *AdminUpdateEventNewsMatchLogic) AdminUpdateEventNewsMatch(req *types.AdminEventNewsMatchUpdateReq) (resp *types.AdminWriteResp, err error) {
	if _, err := utils.GetAdminIDFromCtx(l.ctx); err != nil {
		return &types.AdminWriteResp{Code: 401, Success: false, Message: "未登录或登录已过期"}, nil
	}
	if req == nil || req.MatchId <= 0 || req.EventId <= 0 {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: "请求参数错误"}, nil
	}
	if message := validateAdminEventNewsMatchReq(req.RoundName, req.RoundOrder, req.MatchOrder, req.Status); message != "" {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: message}, nil
	}

	event, err := l.svcCtx.EventNewsModel.FindById(req.EventId)
	if err != nil {
		l.Logger.Errorf("查询赛讯失败: eventId=%d err=%v", req.EventId, err)
		return &types.AdminWriteResp{Code: 500, Success: false, Message: "更新赛事比赛失败"}, nil
	}
	if event == nil || event.TournamentId <= 0 {
		return &types.AdminWriteResp{Code: 404, Success: false, Message: "赛事不存在"}, nil
	}

	match, err := l.svcCtx.TournamentMatchModel.FindById(req.MatchId)
	if err != nil {
		l.Logger.Errorf("查询赛事比赛失败: matchId=%d err=%v", req.MatchId, err)
		return &types.AdminWriteResp{Code: 500, Success: false, Message: "更新赛事比赛失败"}, nil
	}
	if match == nil || match.TournamentId != event.TournamentId {
		return &types.AdminWriteResp{Code: 404, Success: false, Message: "比赛不存在"}, nil
	}

	if err := applyAdminEventNewsMatchUpdate(match, req); err != nil {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: "时间格式不正确"}, nil
	}
	if err := l.svcCtx.TournamentMatchModel.Update(match); err != nil {
		l.Logger.Errorf("更新赛事比赛失败: matchId=%d err=%v", req.MatchId, err)
		return &types.AdminWriteResp{Code: 500, Success: false, Message: "更新赛事比赛失败"}, nil
	}
	return &types.AdminWriteResp{Code: 0, Success: true, Message: "更新成功"}, nil
}
