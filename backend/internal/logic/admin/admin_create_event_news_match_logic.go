package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminCreateEventNewsMatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 创建赛事比赛
func NewAdminCreateEventNewsMatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminCreateEventNewsMatchLogic {
	return &AdminCreateEventNewsMatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *AdminCreateEventNewsMatchLogic) AdminCreateEventNewsMatch(req *types.AdminEventNewsMatchCreateReq) (resp *types.AdminWriteResp, err error) {
	if _, err := utils.GetAdminIDFromCtx(l.ctx); err != nil {
		return &types.AdminWriteResp{Code: 401, Success: false, Message: "未登录或登录已过期"}, nil
	}
	if req == nil || req.EventId <= 0 {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: "请求参数错误"}, nil
	}
	if message := validateAdminEventNewsMatchReq(req.RoundName, req.RoundOrder, req.MatchOrder, req.Status); message != "" {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: message}, nil
	}

	event, err := l.svcCtx.EventNewsModel.FindById(req.EventId)
	if err != nil {
		l.Logger.Errorf("查询赛讯失败: eventId=%d err=%v", req.EventId, err)
		return &types.AdminWriteResp{Code: 500, Success: false, Message: "创建赛事比赛失败"}, nil
	}
	if event == nil || event.TournamentId <= 0 {
		return &types.AdminWriteResp{Code: 404, Success: false, Message: "赛事不存在"}, nil
	}

	match, err := buildAdminEventNewsMatchFromCreate(req, event.TournamentId)
	if err != nil {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: "时间格式不正确"}, nil
	}
	if err := l.svcCtx.TournamentMatchModel.Create(match); err != nil {
		l.Logger.Errorf("创建赛事比赛失败: tournamentId=%d err=%v", event.TournamentId, err)
		return &types.AdminWriteResp{Code: 500, Success: false, Message: "创建赛事比赛失败"}, nil
	}
	return &types.AdminWriteResp{Code: 0, Success: true, Message: "创建成功"}, nil
}
