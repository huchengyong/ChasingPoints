package user

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdatePushTokenLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新推送令牌
func NewUpdatePushTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdatePushTokenLogic {
	return &UpdatePushTokenLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdatePushTokenLogic) UpdatePushToken(req *types.UpdatePushTokenReq) (resp *types.CommonResp, err error) {
	// todo: add your logic here and delete this line

	return
}
