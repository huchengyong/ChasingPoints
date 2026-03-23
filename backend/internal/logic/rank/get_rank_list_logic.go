package rank

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRankListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取段位列表
func NewGetRankListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRankListLogic {
	return &GetRankListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetRankListLogic) GetRankList(req *types.GetRankListReq) (resp *types.GetRankListResp, err error) {
	// todo: add your logic here and delete this line

	return
}
