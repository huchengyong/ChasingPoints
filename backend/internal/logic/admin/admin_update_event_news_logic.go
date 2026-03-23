package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminUpdateEventNewsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新赛事情报
func NewAdminUpdateEventNewsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminUpdateEventNewsLogic {
	return &AdminUpdateEventNewsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminUpdateEventNewsLogic) AdminUpdateEventNews(req *types.AdminEventNewsUpdateReq) (resp *types.AdminWriteResp, err error) {
	// todo: add your logic here and delete this line

	return
}
