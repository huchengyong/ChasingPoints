package admin

import (
	"context"
	"strings"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminCreateEventNewsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 创建赛事情报
func NewAdminCreateEventNewsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminCreateEventNewsLogic {
	return &AdminCreateEventNewsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminCreateEventNewsLogic) AdminCreateEventNews(req *types.AdminEventNewsCreateReq) (resp *types.AdminWriteResp, err error) {
	if _, err := utils.GetAdminIDFromCtx(l.ctx); err != nil {
		return &types.AdminWriteResp{
			Code:    401,
			Success: false,
			Message: "未登录或登录已过期",
		}, nil
	}

	if req == nil {
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
	if sortTime == nil {
		if startTime != nil {
			sortTime = startTime
		} else {
			sortTime = startDate
		}
	}
	if message := validateAdminEventNewsReq(title, req.GameType, req.Status, startDate, endDate); message != "" {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: message,
		}, nil
	}

	endTime, err := parseAdminEventNewsTime(req.EndTime)
	if err != nil {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "结束时间格式不正确",
		}, nil
	}

	now := time.Now()
	tournamentName := fallbackTournamentName(strings.TrimSpace(req.TournamentName), title)
	var tournamentID int64
	if req.TournamentId > 0 {
		tournament, queryErr := l.svcCtx.TournamentModel.FindById(req.TournamentId)
		if queryErr != nil {
			l.Logger.Errorf("查询赛事失败: id=%d err=%v", req.TournamentId, queryErr)
			return &types.AdminWriteResp{Code: 500, Success: false, Message: "创建赛事情报失败"}, nil
		}
		if tournament == nil {
			return &types.AdminWriteResp{Code: 404, Success: false, Message: "赛事不存在"}, nil
		}

		updateReq := &types.AdminEventNewsUpdateReq{
			TournamentId:   req.TournamentId,
			TournamentName: tournamentName,
			Title:          req.Title,
			GameType:       req.GameType,
			Description:    req.Description,
			City:           req.City,
			Venue:          req.Venue,
			StartTime:      req.StartTime,
			EndTime:        req.EndTime,
			Status:         req.Status,
		}
		if err := applyAdminTournamentUpdate(tournament, updateReq); err != nil {
			return &types.AdminWriteResp{Code: 400, Success: false, Message: "时间格式不正确"}, nil
		}
		if message := validateAdminEventNewsTimeRange(tournament.StartDate, tournament.EndDate, tournament.StartTime, tournament.EndTime); message != "" {
			return &types.AdminWriteResp{Code: 400, Success: false, Message: message}, nil
		}
		if err := l.svcCtx.TournamentModel.Update(tournament); err != nil {
			l.Logger.Errorf("更新赛事失败: id=%d err=%v", req.TournamentId, err)
			return &types.AdminWriteResp{Code: 500, Success: false, Message: "创建赛事情报失败"}, nil
		}
		tournamentID = tournament.Id
	} else {
		tournament, buildErr := buildAdminTournament(req)
		if buildErr != nil {
			return &types.AdminWriteResp{Code: 400, Success: false, Message: "时间格式不正确"}, nil
		}
		if message := validateAdminEventNewsTimeRange(tournament.StartDate, tournament.EndDate, tournament.StartTime, tournament.EndTime); message != "" {
			return &types.AdminWriteResp{Code: 400, Success: false, Message: message}, nil
		}
		if err := l.svcCtx.TournamentModel.Create(tournament); err != nil {
			l.Logger.Errorf("创建赛事失败: err=%v", err)
			return &types.AdminWriteResp{Code: 500, Success: false, Message: "创建赛事情报失败"}, nil
		}
		tournamentID = tournament.Id
	}

	news := &model.EventNews{
		Title:        title,
		TournamentId: tournamentID,
		GameType:     req.GameType,
		SourceType:   strings.TrimSpace(req.SourceType),
		SourceName:   strings.TrimSpace(req.SourceName),
		SourceUrl:    strings.TrimSpace(req.SourceUrl),
		CoverImage:   strings.TrimSpace(req.CoverImage),
		Summary:      strings.TrimSpace(req.Summary),
		Content:      req.Content,
		Country:      strings.TrimSpace(req.Country),
		City:         strings.TrimSpace(req.City),
		Venue:        strings.TrimSpace(req.Venue),
		StartDate:    startDate,
		EndDate:      endDate,
		StartTime:    startTime,
		EndTime:      endTime,
		Status:       req.Status,
		SortTime:     sortTime,
		Published:    req.Published,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if req.Published {
		news.PublishedAt = &now
	}

	if err := l.svcCtx.EventNewsModel.Create(news); err != nil {
		l.Logger.Errorf("创建赛事情报失败: err=%v", err)
		return &types.AdminWriteResp{
			Code:    500,
			Success: false,
			Message: "创建赛事情报失败",
		}, nil
	}

	return &types.AdminWriteResp{
		Code:    0,
		Success: true,
		Message: "创建成功",
	}, nil
}
