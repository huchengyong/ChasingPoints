package rules

import (
	"context"
	"strings"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchRulesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 搜索规则
func NewSearchRulesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchRulesLogic {
	return &SearchRulesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SearchRulesLogic) SearchRules(req *types.SearchRulesReq) (resp *types.SearchRulesResp, err error) {
	if strings.TrimSpace(req.Keyword) == "" {
		return &types.SearchRulesResp{Success: true, List: []types.RuleContentItem{}}, nil
	}

	if err = l.svcCtx.RulesContentModel.SeedData(model.GetDefaultRulesContent()); err != nil {
		l.Logger.Errorf("初始化规则搜索数据失败: %v", err)
		return &types.SearchRulesResp{Success: false, List: []types.RuleContentItem{}}, nil
	}

	rows, err := l.svcCtx.RulesContentModel.Search(req.Keyword)
	if err != nil {
		l.Logger.Errorf("搜索规则失败, keyword=%s err=%v", req.Keyword, err)
		return &types.SearchRulesResp{Success: false, List: []types.RuleContentItem{}}, nil
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

	return &types.SearchRulesResp{Success: true, List: list}, nil
}
