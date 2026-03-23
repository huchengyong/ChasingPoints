package challenge

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPendingChallengesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取待处理挑战
func NewGetPendingChallengesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPendingChallengesLogic {
	return &GetPendingChallengesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPendingChallengesLogic) GetPendingChallenges() (resp *types.GetPendingChallengesResp, err error) {
	// todo: add your logic here and delete this line

	return
}
