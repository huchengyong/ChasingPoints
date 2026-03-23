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
		svcCtx: svcCtx,
	}
}

func (l *GetNearbyVenuesLogic) GetNearbyVenues(req *types.GetNearbyVenuesReq) (resp *types.GetVenueListResp, err error) {
	// todo: add your logic here and delete this line

	return
}
