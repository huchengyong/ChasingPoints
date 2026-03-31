package payment

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AlipayMemberSubscriptionNotifyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 支付宝会员支付异步回调
func NewAlipayMemberSubscriptionNotifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AlipayMemberSubscriptionNotifyLogic {
	return &AlipayMemberSubscriptionNotifyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AlipayMemberSubscriptionNotifyLogic) AlipayMemberSubscriptionNotify() (resp *types.CommonResp, err error) {
	gateway := NewGatewayService(l.svcCtx)
	if _, err := gateway.HandleAlipayNotify(l.ctx); err != nil {
		l.Logger.Errorf("处理支付宝会员回调失败: %v", err)
		return &types.CommonResp{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &types.CommonResp{
		Success: true,
		Message: "success",
	}, nil
}
