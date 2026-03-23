package rules

import (
	"context"

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
	// todo: add your logic here and delete this line

	return
}
