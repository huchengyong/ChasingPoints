package venue

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

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
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetNearbyVenuesLogic) GetNearbyVenues(req *types.GetNearbyVenuesReq) (resp *types.GetVenueListResp, err error) {
	if req == nil {
		req = &types.GetNearbyVenuesReq{}
	}
	params := struct {
		Latitude, Longitude float64
		Radius, Limit       int
	}{venueLocationBucket(req.Latitude), venueLocationBucket(req.Longitude), req.Radius, req.Limit}
	result, err := loadVenueCache(l.ctx, l.svcCtx, "nearby", params, venueNearbyCacheTTL, func() (types.GetVenueListResp, error) {
		list, loadErr := l.svcCtx.VenueModel.FindNearbyWithCheckinCount(req.Latitude, req.Longitude, req.Radius, req.Limit)
		if loadErr != nil {
			return types.GetVenueListResp{}, loadErr
		}
		items := make([]types.VenueInfo, 0, len(list))
		for _, venue := range list {
			distance := haversineDistanceMeters(req.Latitude, req.Longitude, venue.Latitude, venue.Longitude)
			items = append(items, buildVenueInfo(venue, int(venue.CheckinCount), distance))
		}
		return types.GetVenueListResp{Success: true, Total: int64(len(items)), List: items}, nil
	})
	if err != nil {
		l.Logger.Errorf("获取附近球馆失败: err=%v", err)
		return &types.GetVenueListResp{Success: false}, nil
	}
	return &result, nil
}
