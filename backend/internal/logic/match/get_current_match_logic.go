package match

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCurrentMatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取进行中对局
func NewGetCurrentMatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCurrentMatchLogic {
	return &GetCurrentMatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCurrentMatchLogic) GetCurrentMatch() (resp *types.GetCurrentMatchResp, err error) {
	// todo: add your logic here and delete this line

	return
}
