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
		svcCtx: svcCtx,
	}
}

func (l *GetVenueDetailLogic) GetVenueDetail(req *types.GetVenueDetailReq) (resp *types.GetVenueDetailResp, err error) {
	// todo: add your logic here and delete this line

	return
}
