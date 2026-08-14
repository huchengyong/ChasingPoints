package rank

import (
	"context"

	"chasing_points/internal/model"
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
		svcCtx: svcCtx.WithContext(ctx),
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

	ranking, err := l.svcCtx.RankingModel.FindByUserIdAndGameType(userId, gameType)
	if err != nil {
		l.Logger.Errorf("获取用户段位失败: %v", err)
		return &types.GetUserRankInfoResp{
			Success: false,
		}, nil
	}
	if ranking == nil {
		ranking = &model.UserRanking{UserId: userId, GameType: gameType, RankLevel: 1}
	}

	configs, err := l.svcCtx.RankingModel.GetAllRankConfigs()
	if err != nil {
		l.Logger.Errorf("获取段位配置失败: %v", err)
		return &types.GetUserRankInfoResp{
			Success: false,
		}, nil
	}
	rankInfo := buildRankInfo(ranking, buildRankConfigByLevel(configs))
	if rankInfo == nil {
		l.Logger.Errorf("获取段位配置失败: 段位等级 %d 不存在", ranking.RankLevel)
		return &types.GetUserRankInfoResp{Success: false}, nil
	}

	return &types.GetUserRankInfoResp{
		Success:  true,
		RankInfo: rankInfo,
	}, nil
}
