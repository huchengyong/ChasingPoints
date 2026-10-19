package stats

import (
	"context"
	"time"

	"chasing_points/internal/ranking"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

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
		svcCtx: svcCtx.WithContext(ctx),
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

	// 排位规则 V2 生效后只展示 V2 下的变化，避免把旧规则的分数与新分数画在同一条线上。
	since := time.Time{}
	if rules := ranking.RulesFor(l.svcCtx.Config); rules.V2ActiveAt(time.Now()) {
		since = rules.V2StartAt()
	}
	logs, err := l.svcCtx.RankingModel.ListRankChangesByUserAndGameTypeSince(userIdInt, gameType, since, limit)
	if err != nil {
		return nil, err
	}

	return &types.GetRankScoreTrendResp{
		Success: true,
		List:    buildTrendPointsFromRankChanges(logs),
	}, nil
}
