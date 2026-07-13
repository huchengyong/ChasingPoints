package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminPublishEventNewsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 发布或下线赛事情报
func NewAdminPublishEventNewsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminPublishEventNewsLogic {
	return &AdminPublishEventNewsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminPublishEventNewsLogic) AdminPublishEventNews(req *types.AdminEventNewsPublishReq) (resp *types.AdminWriteResp, err error) {
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

	existing, err := l.svcCtx.EventNewsModel.FindById(req.EventId)
	if err != nil {
		l.Logger.Errorf("查询赛事情报失败: id=%d err=%v", req.EventId, err)
		return &types.AdminWriteResp{
			Code:    500,
			Success: false,
			Message: "更新发布状态失败",
		}, nil
	}
	if existing == nil {
		return &types.AdminWriteResp{
			Code:    404,
			Success: false,
			Message: "赛事情报不存在",
		}, nil
	}

	if _, err := l.svcCtx.EventNewsModel.UpdatePublished(req.EventId, req.Published); err != nil {
		l.Logger.Errorf("更新赛事情报发布状态失败: id=%d err=%v", req.EventId, err)
		return &types.AdminWriteResp{
			Code:    500,
			Success: false,
			Message: "更新发布状态失败",
		}, nil
	}

	message := "下线成功"
	if req.Published {
		message = "发布成功"
	}

	return &types.AdminWriteResp{
		Code:    0,
		Success: true,
		Message: message,
	}, nil
}
