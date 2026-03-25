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

type AdminUpdateEventNewsStageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新赛事阶段
func NewAdminUpdateEventNewsStageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminUpdateEventNewsStageLogic {
	return &AdminUpdateEventNewsStageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminUpdateEventNewsStageLogic) AdminUpdateEventNewsStage(req *types.AdminEventNewsStageUpdateReq) (resp *types.AdminWriteResp, err error) {
	if _, err := utils.GetAdminIDFromCtx(l.ctx); err != nil {
		return &types.AdminWriteResp{Code: 401, Success: false, Message: "未登录或登录已过期"}, nil
	}
	if req == nil || req.StageId <= 0 {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: "请求参数错误"}, nil
	}
	if message := validateAdminEventNewsStageReq(req.EventId, strings.TrimSpace(req.StageName), req.StageOrder, req.Status); message != "" {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: message}, nil
	}

	event, err := l.svcCtx.EventNewsModel.FindById(req.EventId)
	if err != nil {
		l.Logger.Errorf("查询赛事失败: eventId=%d err=%v", req.EventId, err)
		return &types.AdminWriteResp{Code: 500, Success: false, Message: "更新赛事阶段失败"}, nil
	}
	if event == nil {
		return &types.AdminWriteResp{Code: 404, Success: false, Message: "赛事不存在"}, nil
	}

	stage, err := l.svcCtx.EventNewsStageModel.FindById(req.StageId)
	if err != nil {
		l.Logger.Errorf("查询赛事阶段失败: stageId=%d err=%v", req.StageId, err)
		return &types.AdminWriteResp{Code: 500, Success: false, Message: "更新赛事阶段失败"}, nil
	}
	if stage == nil {
		return &types.AdminWriteResp{Code: 404, Success: false, Message: "赛事阶段不存在"}, nil
	}
	if err := applyAdminEventNewsStageUpdate(stage, req); err != nil {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: "时间格式不正确"}, nil
	}
	if message := validateAdminEventNewsStageTimeRange(stage.StartTime, stage.EndTime); message != "" {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: message}, nil
	}
	if stage.Status < model.EventNewsStatusUpcoming || stage.Status > model.EventNewsStatusCanceled {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: "请选择正确的阶段状态"}, nil
	}

	if err := l.svcCtx.EventNewsStageModel.Update(stage); err != nil {
		l.Logger.Errorf("更新赛事阶段失败: stageId=%d err=%v", req.StageId, err)
		return &types.AdminWriteResp{Code: 500, Success: false, Message: "更新赛事阶段失败"}, nil
	}

	return &types.AdminWriteResp{Code: 0, Success: true, Message: "更新成功"}, nil
}
