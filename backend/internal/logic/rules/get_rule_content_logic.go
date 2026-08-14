package rules

import (
	"context"

	"chasing_points/internal/logic/staticread"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRuleContentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取规则内容
func NewGetRuleContentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRuleContentLogic {
	return &GetRuleContentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetRuleContentLogic) GetRuleContent(req *types.GetRuleContentReq) (resp *types.GetRuleContentResp, err error) {
	category, contentType := "", ""
	if req != nil {
		category, contentType = req.Category, req.ContentType
	}
	result, err := staticread.Load(l.ctx, l.svcCtx, staticread.Key("rules:content", category, contentType), func() (types.GetRuleContentResp, error) {
		rows, loadErr := l.svcCtx.RulesContentModel.FindByCategoryAndType(category, contentType)
		if loadErr != nil {
			return types.GetRuleContentResp{}, loadErr
		}
		list := make([]types.RuleContentItem, 0, len(rows))
		for _, item := range rows {
			list = append(list, types.RuleContentItem{Id: item.Id, Title: item.Title, Content: item.Content, SortOrder: item.SortOrder})
		}
		return types.GetRuleContentResp{Success: true, List: list}, nil
	})
	if err != nil {
		l.Logger.Errorf("查询规则内容失败, category=%s type=%s err=%v", category, contentType, err)
		return &types.GetRuleContentResp{Success: false, List: []types.RuleContentItem{}}, nil
	}
	return &result, nil
}
