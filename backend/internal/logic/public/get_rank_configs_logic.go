package public

import (
	"context"
	"time"

	"chasing_points/internal/logic/staticread"
	"chasing_points/internal/ranking"
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
	// 规则版本随时间切换，不能进入 24 小时的静态缓存，每次按当前时间组装。
	out := cached
	rules := ranking.RulesFor(l.svcCtx.Config)
	out.RulesVersion = rules.VersionAt(time.Now())
	if rules.V2Configured() {
		out.RulesEffectiveAt = rules.V2StartAt().Format(time.RFC3339)
	}
	return &out, nil
}
