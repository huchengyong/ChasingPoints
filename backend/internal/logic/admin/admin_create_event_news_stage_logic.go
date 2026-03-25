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

type AdminCreateEventNewsStageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 创建赛事阶段
func NewAdminCreateEventNewsStageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminCreateEventNewsStageLogic {
	return &AdminCreateEventNewsStageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminCreateEventNewsStageLogic) AdminCreateEventNewsStage(req *types.AdminEventNewsStageCreateReq) (resp *types.AdminWriteResp, err error) {
	if _, err := utils.GetAdminIDFromCtx(l.ctx); err != nil {
		return &types.AdminWriteResp{Code: 401, Success: false, Message: "未登录或登录已过期"}, nil
	}
	if req == nil {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: "请求参数错误"}, nil
	}
	if message := validateAdminEventNewsStageReq(req.EventId, strings.TrimSpace(req.StageName), req.StageOrder, req.Status); message != "" {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: message}, nil
	}

	event, err := l.svcCtx.EventNewsModel.FindById(req.EventId)
	if err != nil {
		l.Logger.Errorf("查询赛事失败: eventId=%d err=%v", req.EventId, err)
		return &types.AdminWriteResp{Code: 500, Success: false, Message: "创建赛事阶段失败"}, nil
	}
	if event == nil {
		return &types.AdminWriteResp{Code: 404, Success: false, Message: "赛事不存在"}, nil
	}

	stage, err := buildAdminEventNewsStage(req)
	if err != nil {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: "时间格式不正确"}, nil
	}
	if message := validateAdminEventNewsStageTimeRange(stage.StartTime, stage.EndTime); message != "" {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: message}, nil
	}
	if stage.Status < model.EventNewsStatusUpcoming || stage.Status > model.EventNewsStatusCanceled {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: "请选择正确的阶段状态"}, nil
	}

	if err := l.svcCtx.EventNewsStageModel.Create(stage); err != nil {
		l.Logger.Errorf("创建赛事阶段失败: eventId=%d err=%v", req.EventId, err)
		return &types.AdminWriteResp{Code: 500, Success: false, Message: "创建赛事阶段失败"}, nil
	}

	return &types.AdminWriteResp{Code: 0, Success: true, Message: "创建成功"}, nil
}
