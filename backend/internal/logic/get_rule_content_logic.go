package logic

import (
	"context"

	"chasing_points/internal/model"
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
		svcCtx: svcCtx,
	}
}

func (l *GetRuleContentLogic) GetRuleContent(req *types.GetRuleContentReq) (resp *types.GetRuleContentResp, err error) {
	if err = l.svcCtx.RulesContentModel.SeedData(model.GetDefaultRulesContent()); err != nil {
		l.Logger.Errorf("初始化规则内容失败: %v", err)
		return &types.GetRuleContentResp{Success: false, List: []types.RuleContentItem{}}, nil
	}

	rows, err := l.svcCtx.RulesContentModel.FindByCategoryAndType(req.Category, req.ContentType)
	if err != nil {
		l.Logger.Errorf("查询规则内容失败, category=%s type=%s err=%v", req.Category, req.ContentType, err)
		return &types.GetRuleContentResp{Success: false, List: []types.RuleContentItem{}}, nil
	}

	list := make([]types.RuleContentItem, 0, len(rows))
	for _, item := range rows {
		list = append(list, types.RuleContentItem{
			Id:        item.Id,
			Title:     item.Title,
			Content:   item.Content,
			SortOrder: item.SortOrder,
		})
	}

	return &types.GetRuleContentResp{Success: true, List: list}, nil
}
