package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminInitLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 初始化管理员账号
func NewAdminInitLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminInitLogic {
	return &AdminInitLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminInitLogic) AdminInit(req *types.AdminInitReq) (resp *types.AdminInitResp, err error) {
	// todo: add your logic here and delete this line

	return
}
