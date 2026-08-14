package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetEventNewsMatchesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取赛事比赛列表
func NewAdminGetEventNewsMatchesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetEventNewsMatchesLogic {
	return &AdminGetEventNewsMatchesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *AdminGetEventNewsMatchesLogic) AdminGetEventNewsMatches(req *types.AdminEventNewsMatchListReq) (resp *types.AdminEventNewsMatchListResp, err error) {
	if _, err := utils.GetAdminIDFromCtx(l.ctx); err != nil {
		return &types.AdminEventNewsMatchListResp{Code: 401, Success: false, Message: "未登录或登录已过期", List: []types.EventNewsMatchInfo{}}, nil
	}
	if req == nil || req.EventId <= 0 {
		return &types.AdminEventNewsMatchListResp{Code: 400, Success: false, Message: "请求参数错误", List: []types.EventNewsMatchInfo{}}, nil
	}

	event, err := l.svcCtx.EventNewsModel.FindById(req.EventId)
	if err != nil {
		l.Logger.Errorf("查询赛讯失败: eventId=%d err=%v", req.EventId, err)
		return &types.AdminEventNewsMatchListResp{Code: 500, Success: false, Message: "获取赛事比赛失败", List: []types.EventNewsMatchInfo{}}, nil
	}
	if event == nil || event.TournamentId <= 0 {
		return &types.AdminEventNewsMatchListResp{Code: 404, Success: false, Message: "赛事不存在", List: []types.EventNewsMatchInfo{}}, nil
	}

	matches, err := l.svcCtx.TournamentMatchModel.FindByTournament(event.TournamentId)
	if err != nil {
		l.Logger.Errorf("查询赛事比赛失败: tournamentId=%d err=%v", event.TournamentId, err)
		return &types.AdminEventNewsMatchListResp{Code: 500, Success: false, Message: "获取赛事比赛失败", List: []types.EventNewsMatchInfo{}}, nil
	}

	items := make([]types.EventNewsMatchInfo, 0, len(matches))
	for _, item := range matches {
		items = append(items, buildAdminEventNewsMatchInfo(event.Id, item))
	}
	if items == nil {
		items = []types.EventNewsMatchInfo{}
	}
	return &types.AdminEventNewsMatchListResp{Code: 0, Success: true, Message: "success", List: items}, nil
}
