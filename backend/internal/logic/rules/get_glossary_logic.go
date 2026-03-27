package rules

import (
	"context"

	"chasing_points/internal/model"
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
		svcCtx: svcCtx,
	}
}

func (l *GetGlossaryLogic) GetGlossary(req *types.GetGlossaryReq) (resp *types.GetGlossaryResp, err error) {
	if err = l.svcCtx.RulesContentModel.SeedData(model.GetDefaultRulesContent()); err != nil {
		l.Logger.Errorf("初始化规则术语失败: %v", err)
		return &types.GetGlossaryResp{Success: false, List: []types.GlossaryItemInfo{}}, nil
	}

	rows, err := l.svcCtx.RulesContentModel.FindGlossary(req.Category)
	if err != nil {
		l.Logger.Errorf("查询术语词典失败, category=%s err=%v", req.Category, err)
		return &types.GetGlossaryResp{Success: false, List: []types.GlossaryItemInfo{}}, nil
	}

	list := make([]types.GlossaryItemInfo, 0, len(rows))
	for _, item := range rows {
		list = append(list, types.GlossaryItemInfo{
			Id:       item.Id,
			Title:    item.Title,
			Content:  item.Content,
			Category: item.Category,
		})
	}

	return &types.GetGlossaryResp{Success: true, List: list}, nil
}
