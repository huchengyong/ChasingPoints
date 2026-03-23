package match

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type MatchScoreLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 加分
func NewMatchScoreLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MatchScoreLogic {
	return &MatchScoreLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MatchScoreLogic) MatchScore(req *types.MatchScoreReq) (resp *types.MatchScoreResp, err error) {
	// todo: add your logic here and delete this line

	return
}
