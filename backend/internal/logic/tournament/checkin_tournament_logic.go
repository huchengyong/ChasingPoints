package tournament

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CheckinTournamentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 签到赛事
func NewCheckinTournamentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckinTournamentLogic {
	return &CheckinTournamentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CheckinTournamentLogic) CheckinTournament(req *types.TournamentIdReq) (resp *types.CommonResp, err error) {
	// todo: add your logic here and delete this line

	return
}
