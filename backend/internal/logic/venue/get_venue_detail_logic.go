package venue

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
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetVenueDetailLogic) GetVenueDetail(req *types.GetVenueDetailReq) (resp *types.GetVenueDetailResp, err error) {
	venueID := int64(0)
	if req != nil {
		venueID = req.VenueId
	}
	result, err := loadVenueCache(l.ctx, l.svcCtx, "detail", venueID, venueDetailCacheTTL, func() (types.GetVenueDetailResp, error) {
		venue, loadErr := l.svcCtx.VenueModel.FindById(venueID)
		if loadErr != nil || venue == nil {
			return types.GetVenueDetailResp{}, loadErr
		}
		checkinCount, loadErr := l.svcCtx.VenueModel.GetCheckinCount(venue.Id)
		if loadErr != nil {
			return types.GetVenueDetailResp{}, loadErr
		}
		item := buildVenueInfo(*venue, int(checkinCount), 0)
		return types.GetVenueDetailResp{Success: true, Venue: &item}, nil
	})
	if err != nil {
		l.Logger.Errorf("获取球馆详情失败: venueId=%d err=%v", venueID, err)
		return &types.GetVenueDetailResp{Success: false}, nil
	}
	return &result, nil
}
