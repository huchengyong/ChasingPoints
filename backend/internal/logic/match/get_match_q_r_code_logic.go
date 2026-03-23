package match

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMatchQRCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取匹配二维码
func NewGetMatchQRCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMatchQRCodeLogic {
	return &GetMatchQRCodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMatchQRCodeLogic) GetMatchQRCode() (resp *types.GetMatchQRCodeResp, err error) {
	// todo: add your logic here and delete this line

	return
}
