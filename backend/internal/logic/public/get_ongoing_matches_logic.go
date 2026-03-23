package public

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOngoingMatchesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取正在进行的对局列表
func NewGetOngoingMatchesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOngoingMatchesLogic {
	return &GetOngoingMatchesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetOngoingMatchesLogic) GetOngoingMatches(req *types.GetOngoingMatchesReq) (resp *types.GetOngoingMatchesResp, err error) {
	// todo: add your logic here and delete this line

	return
}
