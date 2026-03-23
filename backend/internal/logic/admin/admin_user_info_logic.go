package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminUserInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取当前管理员信息
func NewAdminUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminUserInfoLogic {
	return &AdminUserInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminUserInfoLogic) AdminUserInfo() (resp *types.AdminUserInfoResp, err error) {
	// todo: add your logic here and delete this line

	return
}
