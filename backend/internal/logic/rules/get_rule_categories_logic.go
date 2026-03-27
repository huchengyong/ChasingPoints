package rules

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRuleCategoriesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取规则分类
func NewGetRuleCategoriesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRuleCategoriesLogic {
	return &GetRuleCategoriesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetRuleCategoriesLogic) GetRuleCategories() (resp *types.GetRuleCategoriesResp, err error) {
	if err = l.svcCtx.RulesContentModel.SeedData(model.GetDefaultRulesContent()); err != nil {
		l.Logger.Errorf("初始化规则内容失败: %v", err)
		return &types.GetRuleCategoriesResp{Success: false, List: []types.RuleCategoryInfo{}}, nil
	}

	defs := []struct {
		Category    string
		Name        string
		Description string
	}{
		{Category: "chinese_eight", Name: "中式八球", Description: "中式八球规则与术语"},
		{Category: "nine_ball", Name: "九球追分", Description: "九球规则与术语"},
		{Category: "snooker", Name: "斯诺克", Description: "斯诺克规则与术语"},
		{Category: "american_nine", Name: "美式九球", Description: "美式九球规则与术语"},
	}

	list := make([]types.RuleCategoryInfo, 0, len(defs))
	for _, item := range defs {
		rules, findErr := l.svcCtx.RulesContentModel.FindByCategoryAndType(item.Category, "rule")
		if findErr != nil {
			l.Logger.Errorf("查询规则数量失败, category=%s err=%v", item.Category, findErr)
			return &types.GetRuleCategoriesResp{Success: false, List: []types.RuleCategoryInfo{}}, nil
		}

		fouls, findErr := l.svcCtx.RulesContentModel.FindByCategoryAndType(item.Category, "foul")
		if findErr != nil {
			l.Logger.Errorf("查询犯规数量失败, category=%s err=%v", item.Category, findErr)
			return &types.GetRuleCategoriesResp{Success: false, List: []types.RuleCategoryInfo{}}, nil
		}

		glossary, findErr := l.svcCtx.RulesContentModel.FindByCategoryAndType(item.Category, "glossary")
		if findErr != nil {
			l.Logger.Errorf("查询术语数量失败, category=%s err=%v", item.Category, findErr)
			return &types.GetRuleCategoriesResp{Success: false, List: []types.RuleCategoryInfo{}}, nil
		}

		list = append(list, types.RuleCategoryInfo{
			Category:      item.Category,
			Name:          item.Name,
			Description:   item.Description,
			RuleCount:     len(rules),
			FoulCount:     len(fouls),
			GlossaryCount: len(glossary),
		})
	}

	return &types.GetRuleCategoriesResp{Success: true, List: list}, nil
}
