package stats

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOpponentStrengthLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 强弱对手分析
func NewGetOpponentStrengthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOpponentStrengthLogic {
	return &GetOpponentStrengthLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetOpponentStrengthLogic) GetOpponentStrength() (resp *types.GetOpponentStrengthResp, err error) {
	// todo: add your logic here and delete this line

	return
}
