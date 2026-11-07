package admin

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetFeedbackTicketListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取投诉举报与反馈工单列表
func NewAdminGetFeedbackTicketListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetFeedbackTicketListLogic {
	return &AdminGetFeedbackTicketListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *AdminGetFeedbackTicketListLogic) AdminGetFeedbackTicketList(req *types.AdminFeedbackTicketListReq) (resp *types.AdminFeedbackTicketListResp, err error) {
	list, total, err := l.svcCtx.FeedbackTicketModel.FindListForAdmin(model.FeedbackTicketAdminFilter{
		Page:     req.Page,
		PageSize: req.PageSize,
		Status:   req.Status,
		Category: req.Category,
		Source:   req.Source,
	})
	if err != nil {
		l.Logger.Errorf("获取投诉举报工单列表失败: err=%v", err)
		return &types.AdminFeedbackTicketListResp{
			Code:    500,
			Success: false,
			Message: "获取工单列表失败",
		}, nil
	}

	items := make([]types.AdminFeedbackTicketInfo, 0, len(list))
	for _, ticket := range list {
		items = append(items, buildAdminFeedbackTicketInfo(ticket))
	}

	return &types.AdminFeedbackTicketListResp{
		Code:    0,
		Success: true,
		Message: "success",
		Total:   total,
		List:    items,
	}, nil
}
