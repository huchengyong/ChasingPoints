package rules

import (
	"context"

	"chasing_points/internal/logic/staticread"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetGlossaryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取术语词典
func NewGetGlossaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGlossaryLogic {
	return &GetGlossaryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetGlossaryLogic) GetGlossary(req *types.GetGlossaryReq) (resp *types.GetGlossaryResp, err error) {
	category := ""
	if req != nil {
		category = req.Category
	}
	result, err := staticread.Load(l.ctx, l.svcCtx, staticread.Key("rules:glossary", category), func() (types.GetGlossaryResp, error) {
		rows, loadErr := l.svcCtx.RulesContentModel.FindGlossary(category)
		if loadErr != nil {
			return types.GetGlossaryResp{}, loadErr
		}
		list := make([]types.GlossaryItemInfo, 0, len(rows))
		for _, item := range rows {
			list = append(list, types.GlossaryItemInfo{Id: item.Id, Title: item.Title, Content: item.Content, Category: item.Category})
		}
		return types.GetGlossaryResp{Success: true, List: list}, nil
	})
	if err != nil {
		l.Logger.Errorf("查询术语词典失败, category=%s err=%v", category, err)
		return &types.GetGlossaryResp{Success: false, List: []types.GlossaryItemInfo{}}, nil
	}
	return &result, nil
}
