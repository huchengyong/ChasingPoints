package logic

import (
	"context"
	"encoding/json"
	"math"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetVenueListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取球馆列表
func NewGetVenueListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetVenueListLogic {
	return &GetVenueListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetVenueListLogic) GetVenueList(req *types.GetVenueListReq) (resp *types.GetVenueListResp, err error) {
	list, total, err := l.svcCtx.VenueModel.FindList(req.Page, req.PageSize, req.City)
	if err != nil {
		l.Logger.Errorf("获取球馆列表失败: err=%v", err)
		return &types.GetVenueListResp{Success: false}, nil
	}

	items := make([]types.VenueInfo, 0, len(list))
	for _, venue := range list {
		checkinCount, countErr := l.svcCtx.VenueModel.GetCheckinCount(venue.Id)
		if countErr != nil {
			l.Logger.Errorf("获取球馆签到数失败: venueId=%d err=%v", venue.Id, countErr)
		}

		distance := 0.0
		if req.Latitude != 0 && req.Longitude != 0 {
			distance = haversineDistanceMeters(req.Latitude, req.Longitude, venue.Latitude, venue.Longitude)
		}

		items = append(items, buildVenueInfo(venue, int(checkinCount), distance))
	}

	if items == nil {
		items = []types.VenueInfo{}
	}

	return &types.GetVenueListResp{Success: true, Total: total, List: items}, nil
}

func parseVenueImages(imagesJSON string) []string {
	if imagesJSON == "" {
		return []string{}
	}

	var images []string
	if err := json.Unmarshal([]byte(imagesJSON), &images); err != nil {
		return []string{}
	}
	return images
}

func buildVenueInfo(venue model.Venue, checkinCount int, distance float64) types.VenueInfo {
	return types.VenueInfo{
		Id:            venue.Id,
		Name:          venue.Name,
		Address:       venue.Address,
		City:          venue.City,
		District:      venue.District,
		Latitude:      venue.Latitude,
		Longitude:     venue.Longitude,
		Phone:         venue.Phone,
		Images:        parseVenueImages(venue.Images),
		BusinessHours: venue.BusinessHours,
		TableCount:    venue.TableCount,
		PriceRange:    venue.PriceRange,
		Description:   venue.Description,
		Distance:      distance,
		CheckinCount:  checkinCount,
		Status:        venue.Status,
	}
}

func haversineDistanceMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371000.0

	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	dLat := lat2Rad - lat1Rad
	dLon := lon2Rad - lon1Rad

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}
