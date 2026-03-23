package match

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type FinishMatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 结束对局
func NewFinishMatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FinishMatchLogic {
	return &FinishMatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FinishMatchLogic) FinishMatch(req *types.FinishMatchReq) (resp *types.FinishMatchResp, err error) {
	// todo: add your logic here and delete this line

	return
}
