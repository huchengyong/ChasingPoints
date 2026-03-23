package logic

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserRankInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取用户段位信息
func NewGetUserRankInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserRankInfoLogic {
	return &GetUserRankInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserRankInfoLogic) GetUserRankInfo(req *types.GetUserRankInfoReq) (resp *types.GetUserRankInfoResp, err error) {
	// 从上下文获取用户ID
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetUserRankInfoResp{
			Success: false,
		}, nil
	}

	// 获取或创建用户段位信息
	gameType := 3
	if req != nil && req.GameType > 0 {
		gameType = req.GameType
	}

	ranking, err := l.svcCtx.RankingModel.FindOrCreateByGameType(userId, gameType)
	if err != nil {
		l.Logger.Errorf("获取用户段位失败: %v", err)
		return &types.GetUserRankInfoResp{
			Success: false,
		}, nil
	}

	// 获取当前段位配置
	currentConfig, err := l.svcCtx.RankingModel.GetRankConfigByLevel(ranking.RankLevel)
	if err != nil || currentConfig == nil {
		l.Logger.Errorf("获取段位配置失败: %v", err)
		return &types.GetUserRankInfoResp{
			Success: false,
		}, nil
	}

	// 获取下一段位配置
	var nextLevel int
	var nextName string
	var nextScore int
	var progress int

	if ranking.RankLevel < 5 {
		nextConfig, err := l.svcCtx.RankingModel.GetRankConfigByLevel(ranking.RankLevel + 1)
		if err == nil && nextConfig != nil {
			nextLevel = nextConfig.Level
			nextName = nextConfig.Name
			nextScore = nextConfig.MinScore

			// 计算进度百分比
			currentMin := currentConfig.MinScore
			scoreInLevel := ranking.RankScore - currentMin
			scoreToNext := nextScore - currentMin
			if scoreToNext > 0 {
				progress = scoreInLevel * 100 / scoreToNext
				if progress > 100 {
					progress = 100
				}
				if progress < 0 {
					progress = 0
				}
			}
		}
	} else {
		// 已经是最高段位
		nextLevel = 5
		nextName = "钻石王者"
		nextScore = 2000
		progress = 100
	}

	return &types.GetUserRankInfoResp{
		Success: true,
		RankInfo: &types.RankInfo{
			Level:       ranking.RankLevel,
			Name:        currentConfig.Name,
			Icon:        currentConfig.Icon,
			RankScore:   ranking.RankScore,
			TotalWins:   ranking.TotalWins,
			TotalLosses: ranking.TotalLosses,
			MaxStreak:   ranking.MaxStreak,
			NextLevel:   nextLevel,
			NextName:    nextName,
			NextScore:   nextScore,
			Progress:    progress,
		},
	}, nil
}
