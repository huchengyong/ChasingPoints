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
	if startTime == nil && sortTime == nil {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "开始时间和排序时间至少填写一个",
		}, nil
	}
	if sortTime == nil {
		sortTime = startTime
	}
	if message := validateAdminEventNewsReq(title, req.GameType, req.Status, startTime, sortTime); message != "" {
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
	news := &model.EventNews{
		Title:      title,
		GameType:   req.GameType,
		SourceType: strings.TrimSpace(req.SourceType),
		SourceName: strings.TrimSpace(req.SourceName),
		SourceUrl:  strings.TrimSpace(req.SourceUrl),
		CoverImage: strings.TrimSpace(req.CoverImage),
		Summary:    strings.TrimSpace(req.Summary),
		Content:    req.Content,
		Country:    strings.TrimSpace(req.Country),
		City:       strings.TrimSpace(req.City),
		Venue:      strings.TrimSpace(req.Venue),
		StartTime:  startTime,
		EndTime:    endTime,
		Status:     req.Status,
		Featured:   req.Featured,
		SortTime:   sortTime,
		Published:  false,
		CreatedAt:  now,
		UpdatedAt:  now,
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
