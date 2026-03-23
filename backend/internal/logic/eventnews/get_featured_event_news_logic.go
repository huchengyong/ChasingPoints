package eventnews

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFeaturedEventNewsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取首页焦点赛事情报
func NewGetFeaturedEventNewsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFeaturedEventNewsLogic {
	return &GetFeaturedEventNewsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetFeaturedEventNewsLogic) GetFeaturedEventNews() (resp *types.GetFeaturedEventNewsResp, err error) {
	// todo: add your logic here and delete this line

	return
}
