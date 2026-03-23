package rules

import (
	"context"

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
	// todo: add your logic here and delete this line

	return
}
