package eventnews

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetEventNewsListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取赛事情报列表
func NewGetEventNewsListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetEventNewsListLogic {
	return &GetEventNewsListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetEventNewsListLogic) GetEventNewsList(req *types.GetEventNewsListReq) (resp *types.GetEventNewsListResp, err error) {
	// todo: add your logic here and delete this line

	return
}
