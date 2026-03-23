package admin

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
	// todo: add your logic here and delete this line

	return
}
