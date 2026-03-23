package opponent

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOpponentListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取对手列表
func NewGetOpponentListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOpponentListLogic {
	return &GetOpponentListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetOpponentListLogic) GetOpponentList(req *types.GetOpponentListReq) (resp *types.GetOpponentListResp, err error) {
	// todo: add your logic here and delete this line

	return
}
