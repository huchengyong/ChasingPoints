package season

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSeasonLeaderboardLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取赛季排行榜
func NewGetSeasonLeaderboardLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSeasonLeaderboardLogic {
	return &GetSeasonLeaderboardLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSeasonLeaderboardLogic) GetSeasonLeaderboard(req *types.GetSeasonLeaderboardReq) (resp *types.GetSeasonLeaderboardResp, err error) {
	// todo: add your logic here and delete this line

	return
}
