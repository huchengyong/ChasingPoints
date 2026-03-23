package tournament

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type FinishTournamentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 结束赛事
func NewFinishTournamentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FinishTournamentLogic {
	return &FinishTournamentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FinishTournamentLogic) FinishTournament(req *types.TournamentIdReq) (resp *types.CommonResp, err error) {
	// todo: add your logic here and delete this line

	return
}
