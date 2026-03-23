package achievement

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAchievementListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取成就列表
func NewGetAchievementListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAchievementListLogic {
	return &GetAchievementListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAchievementListLogic) GetAchievementList(req *types.GetAchievementListReq) (resp *types.GetAchievementListResp, err error) {
	// todo: add your logic here and delete this line

	return
}
