package tournament

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMyTournamentsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取我的赛事
func NewGetMyTournamentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMyTournamentsLogic {
	return &GetMyTournamentsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMyTournamentsLogic) GetMyTournaments(req *types.GetMyTournamentsReq) (resp *types.GetTournamentListResp, err error) {
	// todo: add your logic here and delete this line

	return
}
