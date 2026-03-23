package opponent

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchOpponentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 搜索对手
func NewSearchOpponentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchOpponentLogic {
	return &SearchOpponentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SearchOpponentLogic) SearchOpponent(req *types.SearchOpponentReq) (resp *types.SearchOpponentResp, err error) {
	// todo: add your logic here and delete this line

	return
}
