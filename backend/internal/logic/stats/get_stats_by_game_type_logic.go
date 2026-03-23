package stats

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetStatsByGameTypeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 分球种统计
func NewGetStatsByGameTypeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetStatsByGameTypeLogic {
	return &GetStatsByGameTypeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetStatsByGameTypeLogic) GetStatsByGameType() (resp *types.GetStatsByGameTypeResp, err error) {
	// todo: add your logic here and delete this line

	return
}
