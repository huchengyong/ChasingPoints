package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminDeleteEventNewsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 删除赛事情报
func NewAdminDeleteEventNewsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminDeleteEventNewsLogic {
	return &AdminDeleteEventNewsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminDeleteEventNewsLogic) AdminDeleteEventNews(req *types.AdminEventNewsIdReq) (resp *types.AdminWriteResp, err error) {
	// todo: add your logic here and delete this line

	return
}
