package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminCreateEventNewsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 创建赛事情报
func NewAdminCreateEventNewsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminCreateEventNewsLogic {
	return &AdminCreateEventNewsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminCreateEventNewsLogic) AdminCreateEventNews(req *types.AdminEventNewsCreateReq) (resp *types.AdminWriteResp, err error) {
	// todo: add your logic here and delete this line

	return
}
