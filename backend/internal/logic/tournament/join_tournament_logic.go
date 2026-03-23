package tournament

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type JoinTournamentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 报名赛事
func NewJoinTournamentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *JoinTournamentLogic {
	return &JoinTournamentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *JoinTournamentLogic) JoinTournament(req *types.TournamentIdReq) (resp *types.CommonResp, err error) {
	// todo: add your logic here and delete this line

	return
}
