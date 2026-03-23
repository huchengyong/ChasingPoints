package venue

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateVenueLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 创建球馆
func NewCreateVenueLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateVenueLogic {
	return &CreateVenueLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateVenueLogic) CreateVenue(req *types.CreateVenueReq) (resp *types.CreateVenueResp, err error) {
	// todo: add your logic here and delete this line

	return
}
