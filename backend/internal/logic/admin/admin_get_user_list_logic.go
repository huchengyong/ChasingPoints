package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetUserListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取用户列表（管理员）
func NewAdminGetUserListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetUserListLogic {
	return &AdminGetUserListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminGetUserListLogic) AdminGetUserList(req *types.AdminUserListReq) (resp *types.AdminUserListResp, err error) {
	// todo: add your logic here and delete this line

	return
}
