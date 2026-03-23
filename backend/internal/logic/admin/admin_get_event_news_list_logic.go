package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetEventNewsListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取赛事情报列表（管理员）
func NewAdminGetEventNewsListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetEventNewsListLogic {
	return &AdminGetEventNewsListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminGetEventNewsListLogic) AdminGetEventNewsList(req *types.AdminEventNewsListReq) (resp *types.AdminEventNewsListResp, err error) {
	// todo: add your logic here and delete this line

	return
}
