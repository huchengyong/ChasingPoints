package achievement

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserAchievementsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取已解锁成就
func NewGetUserAchievementsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserAchievementsLogic {
	return &GetUserAchievementsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserAchievementsLogic) GetUserAchievements() (resp *types.GetUserAchievementsResp, err error) {
	// todo: add your logic here and delete this line

	return
}
