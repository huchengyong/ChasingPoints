package challenge

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RejectChallengeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 拒绝挑战
func NewRejectChallengeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RejectChallengeLogic {
	return &RejectChallengeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RejectChallengeLogic) RejectChallenge(req *types.HandleChallengeReq) (resp *types.CommonResp, err error) {
	// todo: add your logic here and delete this line

	return
}
