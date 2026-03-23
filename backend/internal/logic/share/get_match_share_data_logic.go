package share

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMatchShareDataLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取对局分享数据
func NewGetMatchShareDataLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMatchShareDataLogic {
	return &GetMatchShareDataLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMatchShareDataLogic) GetMatchShareData(req *types.GetMatchShareDataReq) (resp *types.GetMatchShareDataResp, err error) {
	// todo: add your logic here and delete this line

	return
}
