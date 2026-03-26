package logic

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetVenueListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取球馆列表（管理员）
func NewAdminGetVenueListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetVenueListLogic {
	return &AdminGetVenueListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminGetVenueListLogic) AdminGetVenueList(req *types.AdminVenueListReq) (resp *types.AdminVenueListResp, err error) {
	list, total, err := l.svcCtx.VenueModel.FindListForAdmin(req.Page, req.PageSize, req.Status, req.City)
	if err != nil {
		l.Logger.Errorf("获取球馆列表失败: err=%v", err)
		return &types.AdminVenueListResp{
			Code:    500,
			Success: false,
			Message: "获取球馆列表失败",
		}, nil
	}

	items := make([]types.AdminVenueInfo, 0, len(list))
	for _, venue := range list {
		items = append(items, types.AdminVenueInfo{
			Id:            venue.Id,
			Name:          venue.Name,
			Address:       venue.Address,
			City:          venue.City,
			District:      venue.District,
			Latitude:      venue.Latitude,
			Longitude:     venue.Longitude,
			Phone:         venue.Phone,
			Images:        venue.Images,
			BusinessHours: venue.BusinessHours,
			TableCount:    venue.TableCount,
			PriceRange:    venue.PriceRange,
			Description:   venue.Description,
			Status:        venue.Status,
			GeoStatus:     venue.GeoStatus,
			OwnerUserId:   venue.OwnerUserId,
			RejectReason:  venue.RejectReason,
			CreatedAt:     venue.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &types.AdminVenueListResp{
		Code:    0,
		Success: true,
		Message: "success",
		Total:   total,
		List:    items,
	}, nil
}
