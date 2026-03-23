package venue

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

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
		svcCtx: svcCtx,
	}
}

func (l *CheckinVenueLogic) CheckinVenue(req *types.VenueIdReq) (resp *types.CommonResp, err error) {
	// todo: add your logic here and delete this line

	return
}
