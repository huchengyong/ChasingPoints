package venue

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type CheckinVenueLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 打卡球馆
func NewCheckinVenueLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckinVenueLogic {
	return &CheckinVenueLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *CheckinVenueLogic) CheckinVenue(req *types.VenueIdReq) (resp *types.CommonResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CommonResp{Success: false}, nil
	}

	venue, err := l.svcCtx.VenueModel.FindById(req.VenueId)
	if err != nil {
		l.Logger.Errorf("查询球馆失败: venueId=%d err=%v", req.VenueId, err)
		return &types.CommonResp{Success: false, Message: "签到失败"}, nil
	}
	if venue == nil {
		return &types.CommonResp{Success: false, Message: "球馆不存在"}, nil
	}

	hasCheckedIn, err := l.svcCtx.VenueCheckinModel.HasCheckedInToday(req.VenueId, userIdInt)
	if err != nil {
		l.Logger.Errorf("检查签到状态失败: venueId=%d userId=%d err=%v", req.VenueId, userIdInt, err)
		return &types.CommonResp{Success: false, Message: "签到失败"}, nil
	}
	if hasCheckedIn {
		return &types.CommonResp{Success: false, Message: "今日已签到"}, nil
	}

	checkin := &model.VenueCheckin{VenueId: req.VenueId, UserId: userIdInt}
	if err = l.svcCtx.VenueCheckinModel.Create(checkin); err != nil {
		l.Logger.Errorf("创建签到记录失败: venueId=%d userId=%d err=%v", req.VenueId, userIdInt, err)
		return &types.CommonResp{Success: false, Message: "签到失败"}, nil
	}

	return &types.CommonResp{Success: true, Message: "签到成功"}, nil
}
