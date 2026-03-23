package public

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetLeaderboardLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取段位排行榜
func NewGetLeaderboardLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetLeaderboardLogic {
	return &GetLeaderboardLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetLeaderboardLogic) GetLeaderboard(req *types.GetLeaderboardReq) (resp *types.GetLeaderboardResp, err error) {
	// todo: add your logic here and delete this line

	return
}
