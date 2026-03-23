package rules

import (
	"context"

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
	// todo: add your logic here and delete this line

	return
}
