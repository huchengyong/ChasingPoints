package season

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCurrentSeasonLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取当前赛季
func NewGetCurrentSeasonLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCurrentSeasonLogic {
	return &GetCurrentSeasonLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCurrentSeasonLogic) GetCurrentSeason() (resp *types.GetCurrentSeasonResp, err error) {
	// todo: add your logic here and delete this line

	return
}
