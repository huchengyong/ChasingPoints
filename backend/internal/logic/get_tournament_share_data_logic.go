package logic

import (
	"context"

	"billiard_master/internal/svc"
	"billiard_master/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTournamentShareDataLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取赛事分享数据
func NewGetTournamentShareDataLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTournamentShareDataLogic {
	return &GetTournamentShareDataLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTournamentShareDataLogic) GetTournamentShareData(req *types.GetTournamentShareDataReq) (resp *types.GetTournamentShareDataResp, err error) {
	// todo: add your logic here and delete this line

	return
}
