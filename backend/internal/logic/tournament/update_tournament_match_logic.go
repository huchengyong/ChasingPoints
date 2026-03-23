package tournament

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateTournamentMatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新赛事对局结果
func NewUpdateTournamentMatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateTournamentMatchLogic {
	return &UpdateTournamentMatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateTournamentMatchLogic) UpdateTournamentMatch(req *types.UpdateTournamentMatchReq) (resp *types.CommonResp, err error) {
	// todo: add your logic here and delete this line

	return
}
