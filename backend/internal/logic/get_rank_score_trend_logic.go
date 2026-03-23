package logic

import (
	"context"

	"billiard_master/internal/svc"
	"billiard_master/internal/types"
	"billiard_master/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRankScoreTrendLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 段位分变化趋势
func NewGetRankScoreTrendLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRankScoreTrendLogic {
	return &GetRankScoreTrendLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetRankScoreTrendLogic) GetRankScoreTrend(req *types.GetRankScoreTrendReq) (resp *types.GetRankScoreTrendResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetRankScoreTrendResp{Success: false}, nil
	}

	limit := 30
	if req != nil && req.Limit > 0 {
		limit = req.Limit
	}
	gameType := 3
	if req != nil && req.GameType > 0 {
		gameType = req.GameType
	}

	logs, err := l.svcCtx.RankingModel.ListRankChangesByUserAndGameType(userIdInt, gameType, limit)
	if err != nil {
		return nil, err
	}

	return &types.GetRankScoreTrendResp{
		Success: true,
		List:    buildTrendPointsFromRankChanges(logs),
	}, nil
}
