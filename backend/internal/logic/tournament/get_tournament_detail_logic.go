package tournament

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTournamentDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取赛事详情
func NewGetTournamentDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTournamentDetailLogic {
	return &GetTournamentDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTournamentDetailLogic) GetTournamentDetail(req *types.GetTournamentDetailReq) (resp *types.GetTournamentDetailResp, err error) {
	// todo: add your logic here and delete this line

	return
}
