package logic

import (
	"context"

	"billiard_master/internal/svc"
	"billiard_master/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminReviewVenueLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 审核球馆
func NewAdminReviewVenueLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminReviewVenueLogic {
	return &AdminReviewVenueLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminReviewVenueLogic) AdminReviewVenue(req *types.AdminVenueReviewReq) (resp *types.AdminVenueReviewResp, err error) {
	// 验证参数
	if req.VenueId <= 0 {
		return &types.AdminVenueReviewResp{
			Code:    400,
			Success: false,
			Message: "球馆ID无效",
		}, nil
	}

	// 只允许通过(1)或拒绝(3)
	if req.Status != 1 && req.Status != 3 {
		return &types.AdminVenueReviewResp{
			Code:    400,
			Success: false,
			Message: "无效的审核状态",
		}, nil
	}

	// 查询球馆
	venue, err := l.svcCtx.VenueModel.FindById(req.VenueId)
	if err != nil {
		l.Logger.Errorf("查询球馆失败: venueId=%d err=%v", req.VenueId, err)
		return &types.AdminVenueReviewResp{
			Code:    500,
			Success: false,
			Message: "系统错误",
		}, nil
	}
	if venue == nil {
		return &types.AdminVenueReviewResp{
			Code:    404,
			Success: false,
			Message: "球馆不存在",
		}, nil
	}

	// 更新状态
	if err := l.svcCtx.VenueModel.UpdateStatus(req.VenueId, req.Status); err != nil {
		l.Logger.Errorf("更新球馆状态失败: venueId=%d status=%d err=%v", req.VenueId, req.Status, err)
		return &types.AdminVenueReviewResp{
			Code:    500,
			Success: false,
			Message: "审核失败",
		}, nil
	}

	var msg string
	if req.Status == 1 {
		msg = "审核通过"
	} else {
		msg = "审核已拒绝"
	}

	return &types.AdminVenueReviewResp{
		Code:    0,
		Success: true,
		Message: msg,
	}, nil
}
