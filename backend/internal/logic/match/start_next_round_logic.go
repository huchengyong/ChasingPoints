package match

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type StartNextRoundLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 开始下一局
func NewStartNextRoundLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StartNextRoundLogic {
	return &StartNextRoundLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StartNextRoundLogic) StartNextRound(req *types.StartNextRoundReq) (resp *types.StartNextRoundResp, err error) {
	// todo: add your logic here and delete this line

	return
}
