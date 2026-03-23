package match

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type MatchFoulLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 犯规
func NewMatchFoulLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MatchFoulLogic {
	return &MatchFoulLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MatchFoulLogic) MatchFoul(req *types.MatchFoulReq) (resp *types.MatchScoreResp, err error) {
	// todo: add your logic here and delete this line

	return
}
