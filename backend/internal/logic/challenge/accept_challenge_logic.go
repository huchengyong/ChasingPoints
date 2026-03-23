package challenge

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AcceptChallengeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 接受挑战
func NewAcceptChallengeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AcceptChallengeLogic {
	return &AcceptChallengeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AcceptChallengeLogic) AcceptChallenge(req *types.HandleChallengeReq) (resp *types.CommonResp, err error) {
	// todo: add your logic here and delete this line

	return
}
