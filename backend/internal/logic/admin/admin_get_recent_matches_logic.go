package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetRecentMatchesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取最近对局
func NewAdminGetRecentMatchesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetRecentMatchesLogic {
	return &AdminGetRecentMatchesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminGetRecentMatchesLogic) AdminGetRecentMatches() (resp *types.AdminRecentMatchesResp, err error) {
	// todo: add your logic here and delete this line

	return
}
