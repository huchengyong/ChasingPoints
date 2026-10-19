package rank

import (
	"context"
	"time"

	"chasing_points/internal/ranking"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRankListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取段位列表
func NewGetRankListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRankListLogic {
	return &GetRankListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetRankListLogic) GetRankList(req *types.GetRankListReq) (resp *types.GetRankListResp, err error) {
	// 从上下文获取用户ID
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetRankListResp{
			Success: false,
			List:    []types.RankItem{},
		}, nil
	}

	// 获取用户当前段位
	gameType := 3
	if req != nil && req.GameType > 0 {
		gameType = req.GameType
	}

	row, err := l.svcCtx.RankingModel.FindByUserIdAndGameType(userId, gameType)
	if err != nil {
		l.Logger.Errorf("获取用户段位失败: %v", err)
		return &types.GetRankListResp{
			Success: false,
			List:    []types.RankItem{},
		}, nil
	}
	if row == nil {
		row = defaultRankingRow(ranking.RulesFor(l.svcCtx.Config), userId, gameType, time.Now())
	}

	// 获取所有段位配置
	configs, err := l.svcCtx.RankingModel.GetAllRankConfigs()
	if err != nil {
		l.Logger.Errorf("获取段位配置列表失败: %v", err)
		return &types.GetRankListResp{
			Success: false,
			List:    []types.RankItem{},
		}, nil
	}

	// 转换为响应格式
	list := make([]types.RankItem, 0, len(configs))
	for _, config := range configs {
		list = append(list, types.RankItem{
			Level:     config.Level,
			Name:      config.Name,
			Icon:      config.Icon,
			MinScore:  config.MinScore,
			IsCurrent: config.Level == row.RankLevel,
		})
	}

	return &types.GetRankListResp{
		Success: true,
		List:    list,
	}, nil
}
