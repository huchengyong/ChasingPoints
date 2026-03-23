package match

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type StartMatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 开始对局
func NewStartMatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StartMatchLogic {
	return &StartMatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StartMatchLogic) StartMatch(req *types.StartMatchReq) (resp *types.StartMatchResp, err error) {
	// todo: add your logic here and delete this line

	return
}
