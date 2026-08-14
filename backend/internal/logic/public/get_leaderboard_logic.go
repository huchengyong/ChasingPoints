package public

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

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
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetLeaderboardLogic) GetLeaderboard(req *types.GetLeaderboardReq) (resp *types.GetLeaderboardResp, err error) {
	result := &types.GetLeaderboardResp{TopThree: []types.LeaderboardItem{}, List: []types.LeaderboardItem{}}
	if l.svcCtx == nil || l.svcCtx.RankingModel == nil {
		return result, nil
	}
	gameType, page, pageSize := 3, 1, 20
	if req != nil {
		if req.GameType > 0 {
			gameType = req.GameType
		}
		if req.Page > 0 {
			page = req.Page
		}
		if req.PageSize > 0 {
			pageSize = req.PageSize
		}
	}
	if pageSize > 100 {
		pageSize = 100
	}

	shared, err := loadLeaderboardShared(l.ctx, l.svcCtx, "page", gameType, page, pageSize, func() (*leaderboardSharedPayload, error) {
		configs, loadErr := l.svcCtx.RankingModel.GetAllRankConfigs()
		if loadErr != nil {
			return nil, loadErr
		}
		_, rankNames, configVersion := buildRankConfigItems(configs)
		topThree, loadErr := l.svcCtx.RankingModel.GetTopThreeByGameType(gameType)
		if loadErr != nil {
			return nil, loadErr
		}
		topThreeItems := make([]types.LeaderboardItem, 0, len(topThree))
		for index, entry := range topThree {
			topThreeItems = append(topThreeItems, leaderboardItem(entry, index+1, rankNames))
		}
		offset := (page-1)*pageSize + 3
		entries, loadErr := l.svcCtx.RankingModel.GetLeaderboardByGameType(gameType, offset, pageSize)
		if loadErr != nil {
			return nil, loadErr
		}
		list := make([]types.LeaderboardItem, 0, len(entries))
		for index, entry := range entries {
			list = append(list, leaderboardItem(entry, offset+index+1, rankNames))
		}
		total, loadErr := l.svcCtx.RankingModel.GetLeaderboardCountByGameType(gameType)
		if loadErr != nil {
			return nil, loadErr
		}
		return &leaderboardSharedPayload{Total: total, TopThree: topThreeItems, List: list, RankNames: rankNames, ConfigVersion: configVersion}, nil
	})
	if err != nil {
		l.Logger.Errorf("获取共享排行榜失败: %v", err)
		return result, nil
	}

	userID, _ := utils.GetUserIDFromCtx(l.ctx)
	myRanking, rankErr := loadViewerLeaderboardItem(l.svcCtx, userID, gameType, shared.RankNames, true)
	if rankErr != nil {
		l.Logger.Errorf("获取我的排名失败: %v", rankErr)
	}
	result.Success = true
	result.Total = shared.Total
	result.TopThree = shared.TopThree
	result.List = shared.List
	result.MyRanking = myRanking
	return result, nil
}

func loadViewerLeaderboardItem(svcCtx *svc.ServiceContext, userID int64, gameType int, rankNames map[int]string, includeUnranked bool) (*types.LeaderboardItem, error) {
	if svcCtx == nil || svcCtx.RankingModel == nil || userID <= 0 {
		return nil, nil
	}
	rank, entry, err := svcCtx.RankingModel.GetUserRankingByGameType(userID, gameType)
	if err != nil {
		return nil, err
	}
	if entry != nil {
		item := leaderboardItem(*entry, rank, rankNames)
		return &item, nil
	}
	if !includeUnranked || svcCtx.UserModel == nil {
		return nil, nil
	}
	user, err := svcCtx.UserModel.FindById(userID)
	if err != nil || user == nil {
		return nil, err
	}
	return &types.LeaderboardItem{UserId: user.Id, Nickname: user.Nickname, Avatar: user.Avatar, RankLevel: 1, RankName: rankNames[1]}, nil
}
