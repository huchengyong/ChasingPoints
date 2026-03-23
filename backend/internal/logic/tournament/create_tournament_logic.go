package tournament

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateTournamentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 创建赛事
func NewCreateTournamentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateTournamentLogic {
	return &CreateTournamentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateTournamentLogic) CreateTournament(req *types.CreateTournamentReq) (resp *types.CreateTournamentResp, err error) {
	// todo: add your logic here and delete this line

	return
}
