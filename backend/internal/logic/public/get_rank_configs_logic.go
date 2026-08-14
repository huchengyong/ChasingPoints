package public

import (
	"context"

	"chasing_points/internal/logic/staticread"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRankConfigsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取静态段位配置
func NewGetRankConfigsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRankConfigsLogic {
	return &GetRankConfigsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetRankConfigsLogic) GetRankConfigs() (resp *types.GetRankConfigsResp, err error) {
	result := &types.GetRankConfigsResp{List: []types.RankConfigItem{}}
	if l.svcCtx == nil || l.svcCtx.RankingModel == nil {
		return result, nil
	}
	cached, err := staticread.Load(l.ctx, l.svcCtx, "rank-configs", func() (types.GetRankConfigsResp, error) {
		configs, loadErr := l.svcCtx.RankingModel.GetAllRankConfigs()
		if loadErr != nil {
			return types.GetRankConfigsResp{}, loadErr
		}
		items, _, version := buildRankConfigItems(configs)
		return types.GetRankConfigsResp{Success: true, List: items, Version: version}, nil
	})
	if err != nil {
		l.Logger.Errorf("获取段位配置失败: %v", err)
		return result, nil
	}
	return &cached, nil
}
