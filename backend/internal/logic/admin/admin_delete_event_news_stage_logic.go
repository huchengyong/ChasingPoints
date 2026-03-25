package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminDeleteEventNewsStageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 删除赛事阶段
func NewAdminDeleteEventNewsStageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminDeleteEventNewsStageLogic {
	return &AdminDeleteEventNewsStageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminDeleteEventNewsStageLogic) AdminDeleteEventNewsStage(req *types.AdminEventNewsStageIdReq) (resp *types.AdminWriteResp, err error) {
	if _, err := utils.GetAdminIDFromCtx(l.ctx); err != nil {
		return &types.AdminWriteResp{Code: 401, Success: false, Message: "未登录或登录已过期"}, nil
	}
	if req == nil || req.StageId <= 0 {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: "请求参数错误"}, nil
	}

	stage, err := l.svcCtx.EventNewsStageModel.FindById(req.StageId)
	if err != nil {
		l.Logger.Errorf("查询赛事阶段失败: stageId=%d err=%v", req.StageId, err)
		return &types.AdminWriteResp{Code: 500, Success: false, Message: "删除赛事阶段失败"}, nil
	}
	if stage == nil {
		return &types.AdminWriteResp{Code: 404, Success: false, Message: "赛事阶段不存在"}, nil
	}

	if _, err := l.svcCtx.EventNewsStageModel.SoftDelete(req.StageId); err != nil {
		l.Logger.Errorf("删除赛事阶段失败: stageId=%d err=%v", req.StageId, err)
		return &types.AdminWriteResp{Code: 500, Success: false, Message: "删除赛事阶段失败"}, nil
	}

	return &types.AdminWriteResp{Code: 0, Success: true, Message: "删除成功"}, nil
}
