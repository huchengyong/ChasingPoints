package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminChangePasswordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 修改管理员密码
func NewAdminChangePasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminChangePasswordLogic {
	return &AdminChangePasswordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminChangePasswordLogic) AdminChangePassword(req *types.AdminChangePasswordReq) (resp *types.AdminWriteResp, err error) {
	// todo: add your logic here and delete this line

	return
}
