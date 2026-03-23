package admin

import (
	"context"
	"strings"

	"chasing_points/internal/model"
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

	if req == nil || req.EventNewsId <= 0 {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "请求参数错误",
		}, nil
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "请输入标题",
		}, nil
	}
	if req.GameType <= 0 {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "请选择球种",
		}, nil
	}
	if req.Status < model.EventNewsStatusUpcoming || req.Status > model.EventNewsStatusCanceled {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "请选择正确的状态",
		}, nil
	}

	startTimeProvided := strings.TrimSpace(req.StartTime) != ""
	sortTimeProvided := strings.TrimSpace(req.SortTime) != ""
	if !startTimeProvided && !sortTimeProvided {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "开始时间和排序时间至少填写一个",
		}, nil
	}

	existing, err := l.svcCtx.EventNewsModel.FindById(req.EventNewsId)
	if err != nil {
		l.Logger.Errorf("查询赛事情报失败: id=%d err=%v", req.EventNewsId, err)
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

	if err := applyAdminEventNewsPatch(existing, req); err != nil {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "时间格式不正确",
		}, nil
	}
	existing.Title = title
	existing.GameType = req.GameType
	existing.Status = req.Status

	if err := l.svcCtx.EventNewsModel.Update(existing); err != nil {
		l.Logger.Errorf("更新赛事情报失败: id=%d err=%v", req.EventNewsId, err)
		return &types.AdminWriteResp{
			Code:    500,
			Success: false,
			Message: "更新赛事情报失败",
		}, nil
	}

	return &types.AdminWriteResp{
		Code:    0,
		Success: true,
		Message: "更新成功",
	}, nil
}
