package auth

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type BindPhoneLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 绑定手机号
func NewBindPhoneLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BindPhoneLogic {
	return &BindPhoneLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BindPhoneLogic) BindPhone(req *types.BindPhoneReq) (resp *types.BindPhoneResp, err error) {
	// todo: add your logic here and delete this line

	return
}
