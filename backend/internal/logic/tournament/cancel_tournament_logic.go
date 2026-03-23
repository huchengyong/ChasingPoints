package tournament

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelTournamentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 取消赛事
func NewCancelTournamentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelTournamentLogic {
	return &CancelTournamentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CancelTournamentLogic) CancelTournament(req *types.TournamentIdReq) (resp *types.CommonResp, err error) {
	// todo: add your logic here and delete this line

	return
}
