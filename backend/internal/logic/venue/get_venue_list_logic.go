package venue

import (
	"context"

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
	// todo: add your logic here and delete this line

	return
}
