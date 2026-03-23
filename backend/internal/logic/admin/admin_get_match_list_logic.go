package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetMatchListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取对局列表（管理员）
func NewAdminGetMatchListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetMatchListLogic {
	return &AdminGetMatchListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminGetMatchListLogic) AdminGetMatchList(req *types.AdminMatchListReq) (resp *types.AdminMatchListResp, err error) {
	// todo: add your logic here and delete this line

	return
}
