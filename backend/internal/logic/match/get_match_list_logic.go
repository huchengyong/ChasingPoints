package match

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMatchListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取对局列表
func NewGetMatchListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMatchListLogic {
	return &GetMatchListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMatchListLogic) GetMatchList(req *types.MatchListReq) (resp *types.MatchListResp, err error) {
	// todo: add your logic here and delete this line

	return
}
