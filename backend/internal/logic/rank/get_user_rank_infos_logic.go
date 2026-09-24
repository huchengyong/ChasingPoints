package rank

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserRankInfosLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUserRankInfosLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserRankInfosLogic {
	return &GetUserRankInfosLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetUserRankInfosLogic) GetUserRankInfos() (*types.GetUserRankInfosResp, error) {
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetUserRankInfosResp{Success: false}, nil
	}

	rankings, err := l.svcCtx.RankingModel.FindOrCreateByGameTypes(userId)
	if err != nil {
		l.Logger.Errorf("获取用户全部段位失败: %v", err)
		return &types.GetUserRankInfosResp{Success: false}, nil
	}
	configs, err := l.svcCtx.RankingModel.GetAllRankConfigs()
	if err != nil {
		l.Logger.Errorf("获取段位配置失败: %v", err)
		return &types.GetUserRankInfosResp{Success: false}, nil
	}

	byGameType := make(map[int]*types.RankInfo, len(rankings))
	configByLevel := buildRankConfigByLevel(configs)
	for i := range rankings {
		info := buildRankInfo(&rankings[i], configByLevel)
		if info == nil {
			l.Logger.Errorf("段位配置不完整: userId=%d gameType=%d level=%d", userId, rankings[i].GameType, rankings[i].RankLevel)
			return &types.GetUserRankInfosResp{Success: false}, nil
		}
		byGameType[rankings[i].GameType] = info
	}
	gameTypes := model.SupportedRankingGameTypes()
	rankInfos := make([]types.UserRankInfoItem, 0, len(gameTypes))
	for _, gameType := range gameTypes {
		info := byGameType[gameType]
		if info == nil {
			l.Logger.Errorf("用户段位快照不完整: userId=%d gameType=%d", userId, gameType)
			return &types.GetUserRankInfosResp{Success: false}, nil
		}
		rankInfos = append(rankInfos, types.UserRankInfoItem{GameType: gameType, RankInfo: info})
	}
	return &types.GetUserRankInfosResp{Success: true, RankInfos: rankInfos}, nil
}
