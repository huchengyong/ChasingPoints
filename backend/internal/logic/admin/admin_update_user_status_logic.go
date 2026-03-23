package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminUpdateUserStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新用户状态
func NewAdminUpdateUserStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminUpdateUserStatusLogic {
	return &AdminUpdateUserStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminUpdateUserStatusLogic) AdminUpdateUserStatus(req *types.AdminUserUpdateStatusReq) (resp *types.AdminWriteResp, err error) {
	// todo: add your logic here and delete this line

	return
}
