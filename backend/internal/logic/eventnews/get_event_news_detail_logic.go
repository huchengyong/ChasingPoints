package eventnews

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetEventNewsDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取赛事情报详情
func NewGetEventNewsDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetEventNewsDetailLogic {
	return &GetEventNewsDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetEventNewsDetailLogic) GetEventNewsDetail(req *types.GetEventNewsDetailReq) (resp *types.GetEventNewsDetailResp, err error) {
	// todo: add your logic here and delete this line

	return
}
