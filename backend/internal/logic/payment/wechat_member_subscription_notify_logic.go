package payment

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type WechatMemberSubscriptionNotifyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 微信会员支付异步回调
func NewWechatMemberSubscriptionNotifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WechatMemberSubscriptionNotifyLogic {
	return &WechatMemberSubscriptionNotifyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *WechatMemberSubscriptionNotifyLogic) WechatMemberSubscriptionNotify() (resp *types.CommonResp, err error) {
	gateway := NewGatewayService(l.svcCtx)
	if _, err := gateway.HandleWechatNotify(l.ctx); err != nil {
		l.Logger.Errorf("处理微信会员回调失败: %v", err)
		return &types.CommonResp{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &types.CommonResp{
		Success: true,
		Message: "SUCCESS",
	}, nil
}
