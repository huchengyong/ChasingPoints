package member

import (
	"context"
	"strings"

	"chasing_points/internal/config"
	paymentlogic "chasing_points/internal/logic/payment"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMemberSubscriptionOrderStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 查询会员订阅订单状态
func NewGetMemberSubscriptionOrderStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMemberSubscriptionOrderStatusLogic {
	return &GetMemberSubscriptionOrderStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetMemberSubscriptionOrderStatusLogic) GetMemberSubscriptionOrderStatus(req *types.GetMemberSubscriptionOrderStatusReq) (resp *types.GetMemberSubscriptionOrderStatusResp, err error) {
	if !l.svcCtx.Config.MemberPaymentEnabled() {
		return &types.GetMemberSubscriptionOrderStatusResp{
			Success: false,
			Message: config.DisabledFeatureMessage("member_payment"),
		}, nil
	}

	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetMemberSubscriptionOrderStatusResp{
			Success: false,
			Message: "请先完成登录",
		}, nil
	}

	orderNo := strings.TrimSpace(req.OrderNo)
	if orderNo == "" {
		return &types.GetMemberSubscriptionOrderStatusResp{
			Success: false,
			Message: "订单号不能为空",
		}, nil
	}

	order, err := l.svcCtx.MemberSubscriptionOrderModel.FindByOrderNo(orderNo)
	if err != nil {
		l.Logger.Errorf("查询会员订单失败: orderNo=%s err=%v", orderNo, err)
		return &types.GetMemberSubscriptionOrderStatusResp{
			Success: false,
			Message: "查询订单失败",
		}, nil
	}
	if order == nil || order.UserId != userID {
		return &types.GetMemberSubscriptionOrderStatusResp{
			Success: false,
			Message: "订单不存在",
		}, nil
	}

	if order.Status == "pending" {
		gateway := paymentlogic.NewGatewayService(l.svcCtx)
		latestOrder, reconcileErr := gateway.ReconcileOrder(l.ctx, order)
		if reconcileErr != nil {
			l.Logger.Errorf("对账会员订单失败: orderNo=%s err=%v", orderNo, reconcileErr)
		} else if latestOrder != nil {
			order = latestOrder
		}
	}

	resp = paymentlogic.BuildMemberOrderStatusResp(order)
	if resp.Success && resp.Status == "pending" {
		resp.Message = "等待支付结果同步"
	}
	return resp, nil
}
