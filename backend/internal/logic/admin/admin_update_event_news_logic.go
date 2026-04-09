package admin

import (
	"context"
	"strings"

	"chasing_points/internal/logic/eventnews"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminUpdateEventNewsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新赛事情报
func NewAdminUpdateEventNewsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminUpdateEventNewsLogic {
	return &AdminUpdateEventNewsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminUpdateEventNewsLogic) AdminUpdateEventNews(req *types.AdminEventNewsUpdateReq) (resp *types.AdminWriteResp, err error) {
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

	title := strings.TrimSpace(req.Title)
	startDate, err := parseAdminEventNewsDate(req.StartDate)
	if err != nil {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "开始日期格式不正确",
		}, nil
	}
	endDate, err := parseAdminEventNewsDate(req.EndDate)
	if err != nil {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "结束日期格式不正确",
		}, nil
	}
	if endDate == nil {
		endDate = startDate
	}
	startTime, err := parseAdminEventNewsTime(req.StartTime)
	if err != nil {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "开始时间格式不正确",
		}, nil
	}
	sortTime, err := parseAdminEventNewsTime(req.SortTime)
	if err != nil {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "排序时间格式不正确",
		}, nil
	}
	if sortTime == nil && startTime != nil {
		sortTime = startTime
	}
	if sortTime == nil {
		sortTime = startDate
	}
	if message := validateAdminEventNewsReq(title, req.GameType, req.Status, startDate, endDate); message != "" {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: message,
		}, nil
	}

	existing, err := l.svcCtx.EventNewsModel.FindById(req.EventId)
	if err != nil {
		l.Logger.Errorf("查询赛事情报失败: id=%d err=%v", req.EventId, err)
		return &types.AdminWriteResp{
			Code:    500,
			Success: false,
			Message: "更新赛事情报失败",
		}, nil
	}
	if existing == nil {
		return &types.AdminWriteResp{
			Code:    404,
			Success: false,
			Message: "赛事情报不存在",
		}, nil
	}

	if err := applyAdminEventNewsUpdate(existing, req); err != nil {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "时间格式不正确",
		}, nil
	}
	effectiveTournamentID := existing.TournamentId
	if req.TournamentId > 0 {
		effectiveTournamentID = req.TournamentId
	}

	existing.Title = title
	existing.TournamentId = effectiveTournamentID
	existing.GameType = req.GameType
	existing.Status = req.Status

	if effectiveTournamentID > 0 {
		tournament, queryErr := l.svcCtx.TournamentModel.FindById(effectiveTournamentID)
		if queryErr != nil {
			l.Logger.Errorf("查询赛事失败: tournamentId=%d err=%v", effectiveTournamentID, queryErr)
			return &types.AdminWriteResp{Code: 500, Success: false, Message: "更新赛事情报失败"}, nil
		}
		if tournament == nil {
			return &types.AdminWriteResp{Code: 404, Success: false, Message: "赛事不存在"}, nil
		}
		if err := applyAdminTournamentUpdate(tournament, req); err != nil {
			return &types.AdminWriteResp{Code: 400, Success: false, Message: "时间格式不正确"}, nil
		}
		if message := validateAdminEventNewsTimeRange(tournament.StartDate, tournament.EndDate, tournament.StartTime, tournament.EndTime); message != "" {
			return &types.AdminWriteResp{Code: 400, Success: false, Message: message}, nil
		}
		if err := l.svcCtx.TournamentModel.Update(tournament); err != nil {
			l.Logger.Errorf("更新赛事失败: tournamentId=%d err=%v", effectiveTournamentID, err)
			return &types.AdminWriteResp{Code: 500, Success: false, Message: "更新赛事情报失败"}, nil
		}
	}

	if err := l.svcCtx.EventNewsModel.Update(existing); err != nil {
		l.Logger.Errorf("更新赛事情报失败: id=%d err=%v", req.EventId, err)
		return &types.AdminWriteResp{
			Code:    500,
			Success: false,
			Message: "更新赛事情报失败",
		}, nil
	}
	if err := eventnews.BumpEventNewsCacheVersion(l.ctx, l.svcCtx); err != nil {
		l.Logger.Errorf("更新赛事情报后更新缓存版本失败: eventId=%d err=%v", existing.Id, err)
	}

	return &types.AdminWriteResp{
		Code:    0,
		Success: true,
		Message: "更新成功",
	}, nil
}
