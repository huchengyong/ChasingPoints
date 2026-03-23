package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetRecentUsersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取最近注册用户
func NewAdminGetRecentUsersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetRecentUsersLogic {
	return &AdminGetRecentUsersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminGetRecentUsersLogic) AdminGetRecentUsers() (resp *types.AdminRecentUsersResp, err error) {
	// todo: add your logic here and delete this line

	return
}
