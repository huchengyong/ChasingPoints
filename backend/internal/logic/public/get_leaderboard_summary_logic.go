package public

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetLeaderboardSummaryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取榜单前三名与当前用户排名
func NewGetLeaderboardSummaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetLeaderboardSummaryLogic {
	return &GetLeaderboardSummaryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetLeaderboardSummaryLogic) GetLeaderboardSummary(req *types.GetLeaderboardSummaryReq) (resp *types.GetLeaderboardSummaryResp, err error) {
	result := &types.GetLeaderboardSummaryResp{TopThree: []types.LeaderboardItem{}}
	if l.svcCtx == nil || l.svcCtx.RankingModel == nil {
		return result, nil
	}
	gameType := 3
	if req != nil && req.GameType > 0 {
		gameType = req.GameType
	}
	shared, err := loadLeaderboardShared(l.ctx, l.svcCtx, "summary", gameType, 0, 3, func() (*leaderboardSharedPayload, error) {
		configs, loadErr := l.svcCtx.RankingModel.GetAllRankConfigs()
		if loadErr != nil {
			return nil, loadErr
		}
		_, rankNames, version := buildRankConfigItems(configs)
		topThree, loadErr := l.svcCtx.RankingModel.GetTopThreeByGameType(gameType)
		if loadErr != nil {
			return nil, loadErr
		}
		items := make([]types.LeaderboardItem, 0, len(topThree))
		for index, entry := range topThree {
			items = append(items, leaderboardItem(entry, index+1, rankNames))
		}
		return &leaderboardSharedPayload{TopThree: items, RankNames: rankNames, ConfigVersion: version}, nil
	})
	if err != nil {
		l.Logger.Errorf("获取共享榜单摘要失败: %v", err)
		return result, nil
	}
	userID, _ := utils.GetUserIDFromCtx(l.ctx)
	myRanking, rankErr := loadViewerLeaderboardItem(l.svcCtx, userID, gameType, shared.RankNames, false)
	if rankErr != nil {
		l.Logger.Errorf("获取当前用户排名失败: %v", rankErr)
	}
	result.Success = true
	result.TopThree = shared.TopThree
	result.MyRanking = myRanking
	result.Version = shared.ConfigVersion
	return result, nil
}
