package tournament

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTournamentBracketLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取赛事对阵图
func NewGetTournamentBracketLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTournamentBracketLogic {
	return &GetTournamentBracketLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTournamentBracketLogic) GetTournamentBracket(req *types.GetTournamentBracketReq) (resp *types.GetTournamentBracketResp, err error) {
	// todo: add your logic here and delete this line

	return
}
