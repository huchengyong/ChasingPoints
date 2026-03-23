package public

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPublicMatchDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取公开对局详情
func NewGetPublicMatchDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPublicMatchDetailLogic {
	return &GetPublicMatchDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPublicMatchDetailLogic) GetPublicMatchDetail(req *types.GetPublicMatchDetailReq) (resp *types.GetPublicMatchDetailResp, err error) {
	// todo: add your logic here and delete this line

	return
}
