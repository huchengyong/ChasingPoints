package logic

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
		svcCtx: svcCtx,
	}
}

func (l *GetLeaderboardLogic) GetLeaderboard(req *types.GetLeaderboardReq) (resp *types.GetLeaderboardResp, err error) {
	// 尝试从上下文获取用户ID（公开接口，未登录时可能获取不到）
	userId, _ := utils.GetUserIDFromCtx(l.ctx)
	gameType := 3
	if req != nil && req.GameType > 0 {
		gameType = req.GameType
	}

	// 获取所有段位配置用于映射段位名称
	configs, err := l.svcCtx.RankingModel.GetAllRankConfigs()
	if err != nil {
		l.Logger.Errorf("获取段位配置失败: %v", err)
		return &types.GetLeaderboardResp{
			Success:  false,
			TopThree: []types.LeaderboardItem{},
			List:     []types.LeaderboardItem{},
		}, nil
	}
	rankNameMap := make(map[int]string)
	for _, config := range configs {
		rankNameMap[config.Level] = config.Name
	}

	// 获取前三名
	topThree, err := l.svcCtx.RankingModel.GetTopThreeByGameType(gameType)
	if err != nil {
		l.Logger.Errorf("获取前三名失败: %v", err)
		return &types.GetLeaderboardResp{
			Success:  false,
			TopThree: []types.LeaderboardItem{},
			List:     []types.LeaderboardItem{},
		}, nil
	}

	// 转换前三名数据
	topThreeItems := make([]types.LeaderboardItem, 0, len(topThree))
	for i, entry := range topThree {
		winRate := 0
		if entry.TotalWins+entry.TotalLosses > 0 {
			winRate = entry.TotalWins * 100 / (entry.TotalWins + entry.TotalLosses)
		}
		topThreeItems = append(topThreeItems, types.LeaderboardItem{
			Rank:      i + 1,
			UserId:    entry.UserId,
			Nickname:  entry.Nickname,
			Avatar:    entry.Avatar,
			RankLevel: entry.RankLevel,
			RankName:  rankNameMap[entry.RankLevel],
			RankScore: entry.RankScore,
			WinRate:   winRate,
		})
	}

	// 获取我的排名
	myRank, myEntry, err := l.svcCtx.RankingModel.GetUserRankingByGameType(userId, gameType)
	if err != nil {
		l.Logger.Errorf("获取我的排名失败: %v", err)
	}
	var myRanking *types.LeaderboardItem
	if myEntry != nil {
		winRate := 0
		if myEntry.TotalWins+myEntry.TotalLosses > 0 {
			winRate = myEntry.TotalWins * 100 / (myEntry.TotalWins + myEntry.TotalLosses)
		}
		myRanking = &types.LeaderboardItem{
			Rank:      myRank,
			UserId:    myEntry.UserId,
			Nickname:  myEntry.Nickname,
			Avatar:    myEntry.Avatar,
			RankLevel: myEntry.RankLevel,
			RankName:  rankNameMap[myEntry.RankLevel],
			RankScore: myEntry.RankScore,
			WinRate:   winRate,
		}
	} else {
		// 用户没有排名记录，从 users 表获取基本信息
		user, userErr := l.svcCtx.UserModel.FindById(userId)
		if userErr == nil && user != nil {
			myRanking = &types.LeaderboardItem{
				Rank:      0, // 0 表示未上榜，前端显示为 "-"
				UserId:    user.Id,
				Nickname:  user.Nickname,
				Avatar:    user.Avatar,
				RankLevel: 1, // 默认青铜
				RankName:  rankNameMap[1],
				RankScore: 0,
				WinRate:   0,
			}
		}
	}

	// 计算偏移量（跳过前三名）
	offset := (req.Page-1)*req.PageSize + 3
	if req.Page == 1 {
		offset = 3 // 第一页从第4名开始
	}

	// 获取排行榜列表（第4名开始）
	entries, err := l.svcCtx.RankingModel.GetLeaderboardByGameType(gameType, offset, req.PageSize)
	if err != nil {
		l.Logger.Errorf("获取排行榜列表失败: %v", err)
		return &types.GetLeaderboardResp{
			Success:   false,
			TopThree:  topThreeItems,
			MyRanking: myRanking,
			List:      []types.LeaderboardItem{},
		}, nil
	}

	// 转换列表数据
	list := make([]types.LeaderboardItem, 0, len(entries))
	for i, entry := range entries {
		winRate := 0
		if entry.TotalWins+entry.TotalLosses > 0 {
			winRate = entry.TotalWins * 100 / (entry.TotalWins + entry.TotalLosses)
		}
		list = append(list, types.LeaderboardItem{
			Rank:      offset + i + 1,
			UserId:    entry.UserId,
			Nickname:  entry.Nickname,
			Avatar:    entry.Avatar,
			RankLevel: entry.RankLevel,
			RankName:  rankNameMap[entry.RankLevel],
			RankScore: entry.RankScore,
			WinRate:   winRate,
		})
	}

	// 获取总人数
	total, err := l.svcCtx.RankingModel.GetLeaderboardCountByGameType(gameType)
	if err != nil {
		l.Logger.Errorf("获取排行榜总人数失败: %v", err)
		total = 0
	}

	return &types.GetLeaderboardResp{
		Success:   true,
		Total:     total,
		TopThree:  topThreeItems,
		MyRanking: myRanking,
		List:      list,
	}, nil
}
