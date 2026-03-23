package tournament

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type LeaveTournamentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 退出赛事
func NewLeaveTournamentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LeaveTournamentLogic {
	return &LeaveTournamentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LeaveTournamentLogic) LeaveTournament(req *types.TournamentIdReq) (resp *types.CommonResp, err error) {
	// todo: add your logic here and delete this line

	return
}
