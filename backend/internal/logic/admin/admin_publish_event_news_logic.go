package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminPublishEventNewsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 发布或下线赛事情报
func NewAdminPublishEventNewsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminPublishEventNewsLogic {
	return &AdminPublishEventNewsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminPublishEventNewsLogic) AdminPublishEventNews(req *types.AdminEventNewsIdReq) (resp *types.AdminWriteResp, err error) {
	// todo: add your logic here and delete this line

	return
}
