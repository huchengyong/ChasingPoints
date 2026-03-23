package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

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
	// todo: add your logic here and delete this line

	return
}
