package tournament

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTournamentListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取赛事列表
func NewGetTournamentListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTournamentListLogic {
	return &GetTournamentListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTournamentListLogic) GetTournamentList(req *types.GetTournamentListReq) (resp *types.GetTournamentListResp, err error) {
	// todo: add your logic here and delete this line

	return
}
