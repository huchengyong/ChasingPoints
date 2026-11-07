package admin

import (
	"context"
	"errors"
	"strings"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type AdminProcessFeedbackTicketLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 处理投诉举报与反馈工单
func NewAdminProcessFeedbackTicketLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminProcessFeedbackTicketLogic {
	return &AdminProcessFeedbackTicketLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *AdminProcessFeedbackTicketLogic) AdminProcessFeedbackTicket(req *types.AdminFeedbackTicketProcessReq) (resp *types.AdminWriteResp, err error) {
	adminID, err := utils.GetAdminIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取管理员ID失败: %v", err)
		return &types.AdminWriteResp{Code: 401, Success: false, Message: "获取管理员信息失败"}, nil
	}
	if req.TicketId <= 0 {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: "工单ID无效"}, nil
	}
	if !model.IsFeedbackTicketProcessStatus(req.Status) {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: "无效的处理状态"}, nil
	}

	processResult := strings.TrimSpace(req.ProcessResult)
	if req.Status == model.FeedbackTicketStatusProcessing && processResult == "" {
		processResult = "已受理"
	}
	if (req.Status == model.FeedbackTicketStatusResolved || req.Status == model.FeedbackTicketStatusClosed) && processResult == "" {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: "请填写处理结果"}, nil
	}

	if err := l.svcCtx.FeedbackTicketModel.Process(req.TicketId, req.Status, processResult, int64(adminID), time.Now()); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &types.AdminWriteResp{Code: 404, Success: false, Message: "工单不存在"}, nil
		}
		l.Logger.Errorf("处理投诉举报工单失败: ticketId=%d status=%d err=%v", req.TicketId, req.Status, err)
		return &types.AdminWriteResp{Code: 500, Success: false, Message: "处理失败"}, nil
	}

	return &types.AdminWriteResp{Code: 0, Success: true, Message: "处理成功"}, nil
}
