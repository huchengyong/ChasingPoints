package member

import (
	"context"

	"chasing_points/internal/config"
	paymentlogic "chasing_points/internal/logic/payment"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateMemberSubscriptionOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 创建会员订阅订单
func NewCreateMemberSubscriptionOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateMemberSubscriptionOrderLogic {
	return &CreateMemberSubscriptionOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *CreateMemberSubscriptionOrderLogic) CreateMemberSubscriptionOrder(req *types.CreateMemberSubscriptionOrderReq) (resp *types.CreateMemberSubscriptionOrderResp, err error) {
	// 合规收口：上线前暂时关闭会员订阅付费，待取得相关资质或完成专项合规评估后再恢复。
	if !l.svcCtx.Config.MemberPaymentEnabled() {
		return &types.CreateMemberSubscriptionOrderResp{
			Success: false,
			Message: config.DisabledFeatureMessage("member_payment"),
		}, nil
	}

	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CreateMemberSubscriptionOrderResp{
			Success: false,
			Message: "请先完成登录",
		}, nil
	}

	user, err := l.svcCtx.UserModel.FindById(userID)
	if err != nil {
		l.Logger.Errorf("查询用户失败: userId=%d err=%v", userID, err)
		return &types.CreateMemberSubscriptionOrderResp{
			Success: false,
			Message: "获取用户信息失败",
		}, nil
	}
	if user == nil {
		return &types.CreateMemberSubscriptionOrderResp{
			Success: false,
			Message: "用户不存在",
		}, nil
	}

	plan, ok := paymentlogic.FindMemberPlan(req.PlanCode)
	if !ok {
		return &types.CreateMemberSubscriptionOrderResp{
			Success: false,
			Message: "会员套餐不存在",
		}, nil
	}

	payChannel, ok := paymentlogic.NormalizePayChannel(req.PayChannel)
	if !ok {
		return &types.CreateMemberSubscriptionOrderResp{
			Success: false,
			Message: "支付方式不支持",
		}, nil
	}

	order := &model.MemberSubscriptionOrder{
		OrderNo:      paymentlogic.GenerateMemberSubscriptionOrderNo(userID),
		UserId:       userID,
		PlanCode:     plan.Code,
		PlanName:     plan.Name,
		DurationDays: plan.DurationDays,
		AmountFen:    plan.PriceFen,
		PayChannel:   payChannel,
		Status:       model.MemberSubscriptionOrderStatusPending,
	}
	if err := l.svcCtx.MemberSubscriptionOrderModel.Create(order); err != nil {
		l.Logger.Errorf("创建会员订单失败: userId=%d err=%v", userID, err)
		return &types.CreateMemberSubscriptionOrderResp{
			Success: false,
			Message: "创建订单失败，请稍后重试",
		}, nil
	}

	gateway := paymentlogic.NewGatewayService(l.svcCtx)
	alipayOrderString, wechatAppPayParams, err := gateway.CreatePayPayload(l.ctx, order)
	if err != nil {
		l.Logger.Errorf("生成支付参数失败: orderNo=%s err=%v", order.OrderNo, err)
		return &types.CreateMemberSubscriptionOrderResp{
			Success:    false,
			Message:    "生成支付参数失败，请稍后重试",
			OrderNo:    order.OrderNo,
			PayChannel: payChannel,
			PlanCode:   plan.Code,
			PlanName:   plan.Name,
			AmountFen:  plan.PriceFen,
			AmountYuan: paymentlogic.FormatAmountFenToYuan(plan.PriceFen),
			Status:     order.Status,
		}, nil
	}

	return &types.CreateMemberSubscriptionOrderResp{
		Success:            true,
		Message:            "下单成功",
		OrderNo:            order.OrderNo,
		PayChannel:         payChannel,
		PlanCode:           plan.Code,
		PlanName:           plan.Name,
		AmountFen:          plan.PriceFen,
		AmountYuan:         paymentlogic.FormatAmountFenToYuan(plan.PriceFen),
		Status:             order.Status,
		AlipayOrderString:  alipayOrderString,
		WechatAppPayParams: wechatAppPayParams,
	}, nil
}
