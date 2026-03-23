package logic

import (
	"context"

	"billiard_master/internal/svc"
	"billiard_master/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetNearbyVenuesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取附近球馆
func NewGetNearbyVenuesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetNearbyVenuesLogic {
	return &GetNearbyVenuesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetNearbyVenuesLogic) GetNearbyVenues(req *types.GetNearbyVenuesReq) (resp *types.GetVenueListResp, err error) {
	list, err := l.svcCtx.VenueModel.FindNearby(req.Latitude, req.Longitude, req.Radius, req.Limit)
	if err != nil {
		l.Logger.Errorf("获取附近球馆失败: err=%v", err)
		return &types.GetVenueListResp{Success: false}, nil
	}

	items := make([]types.VenueInfo, 0, len(list))
	for _, venue := range list {
		checkinCount, countErr := l.svcCtx.VenueModel.GetCheckinCount(venue.Id)
		if countErr != nil {
			l.Logger.Errorf("获取球馆签到数失败: venueId=%d err=%v", venue.Id, countErr)
		}

		distance := haversineDistanceMeters(req.Latitude, req.Longitude, venue.Latitude, venue.Longitude)
		items = append(items, buildVenueInfo(venue, int(checkinCount), distance))
	}

	if items == nil {
		items = []types.VenueInfo{}
	}

	return &types.GetVenueListResp{Success: true, Total: int64(len(items)), List: items}, nil
}
