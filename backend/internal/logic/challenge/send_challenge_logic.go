package challenge

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendChallengeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 发起挑战
func NewSendChallengeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendChallengeLogic {
	return &SendChallengeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SendChallengeLogic) SendChallenge(req *types.SendChallengeReq) (resp *types.SendChallengeResp, err error) {
	// todo: add your logic here and delete this line

	return
}
