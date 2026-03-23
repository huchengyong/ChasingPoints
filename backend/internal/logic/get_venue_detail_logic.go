package logic

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetVenueDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取球馆详情
func NewGetVenueDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetVenueDetailLogic {
	return &GetVenueDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetVenueDetailLogic) GetVenueDetail(req *types.GetVenueDetailReq) (resp *types.GetVenueDetailResp, err error) {
	venue, err := l.svcCtx.VenueModel.FindById(req.VenueId)
	if err != nil {
		l.Logger.Errorf("获取球馆详情失败: venueId=%d err=%v", req.VenueId, err)
		return &types.GetVenueDetailResp{Success: false}, nil
	}
	if venue == nil {
		return &types.GetVenueDetailResp{Success: false}, nil
	}

	checkinCount, err := l.svcCtx.VenueModel.GetCheckinCount(venue.Id)
	if err != nil {
		l.Logger.Errorf("获取球馆签到数失败: venueId=%d err=%v", venue.Id, err)
		return &types.GetVenueDetailResp{Success: false}, nil
	}

	item := buildVenueInfo(*venue, int(checkinCount), 0)
	return &types.GetVenueDetailResp{Success: true, Venue: &item}, nil
}
