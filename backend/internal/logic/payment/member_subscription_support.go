package payment

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/go-pay/gopay"
	gopayalipay "github.com/go-pay/gopay/alipay"
	gopaywechat "github.com/go-pay/gopay/wechat/v3"
)

const (
	PayChannelAlipay = "alipay"
	PayChannelWechat = "wechat"

	memberMonthlyPlanCode = "member_monthly"
)

type MemberPlan struct {
	Code         string
	Name         string
	PriceFen     int
	DurationDays int
	Description  string
	Highlight    string
}

type GatewayService struct {
	svcCtx *svc.ServiceContext
}

func NewGatewayService(svcCtx *svc.ServiceContext) *GatewayService {
	return &GatewayService{svcCtx: svcCtx}
}

func MemberPlans() []MemberPlan {
	return []MemberPlan{
		{
			Code:         memberMonthlyPlanCode,
			Name:         "月卡会员",
			PriceFen:     1900,
			DurationDays: 30,
			Description:  "APP 端月卡订阅，支付成功后立即生效。",
			Highlight:    "首发月卡",
		},
	}
}

func FindMemberPlan(planCode string) (MemberPlan, bool) {
	normalized := strings.TrimSpace(planCode)
	for _, item := range MemberPlans() {
		if item.Code == normalized {
			return item, true
		}
	}
	return MemberPlan{}, false
}

func NormalizePayChannel(channel string) (string, bool) {
	switch strings.TrimSpace(strings.ToLower(channel)) {
	case PayChannelAlipay:
		return PayChannelAlipay, true
	case PayChannelWechat:
		return PayChannelWechat, true
	default:
		return "", false
	}
}

func BuildMemberPlanInfo(plan MemberPlan) types.MemberPlanInfo {
	return types.MemberPlanInfo{
		PlanCode:      plan.Code,
		PlanName:      plan.Name,
		PriceFen:      plan.PriceFen,
		PriceYuan:     FormatAmountFenToYuan(plan.PriceFen),
		DurationDays:  plan.DurationDays,
		DurationLabel: fmt.Sprintf("%d 天", plan.DurationDays),
		Description:   plan.Description,
		Highlight:     plan.Highlight,
	}
}

func FormatAmountFenToYuan(amountFen int) string {
	sign := ""
	value := amountFen
	if value < 0 {
		sign = "-"
		value = -value
	}
	return fmt.Sprintf("%s%d.%02d", sign, value/100, value%100)
}

func ParseAmountYuanToFen(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, errors.New("amount is empty")
	}

	sign := 1
	if strings.HasPrefix(value, "-") {
		sign = -1
		value = strings.TrimPrefix(value, "-")
	}

	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid amount: %s", raw)
	}

	integerPart := parts[0]
	if integerPart == "" {
		integerPart = "0"
	}
	integerValue, err := strconv.Atoi(integerPart)
	if err != nil {
		return 0, fmt.Errorf("invalid amount integer: %w", err)
	}

	decimalValue := 0
	if len(parts) == 2 {
		decimalPart := parts[1]
		if len(decimalPart) > 2 {
			return 0, fmt.Errorf("invalid amount precision: %s", raw)
		}
		if len(decimalPart) == 1 {
			decimalPart += "0"
		}
		if decimalPart != "" {
			decimalValue, err = strconv.Atoi(decimalPart)
			if err != nil {
				return 0, fmt.Errorf("invalid amount decimal: %w", err)
			}
		}
	}

	return sign * (integerValue*100 + decimalValue), nil
}

func ParseUTC8Time(raw string) (time.Time, error) {
	return time.ParseInLocation(logicx.UTC8Layout, strings.TrimSpace(raw), logicx.UTC8Location)
}

func ParseRFC3339ToUTC8(raw string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, err
	}
	return parsed.In(logicx.UTC8Location), nil
}

func GenerateMemberSubscriptionOrderNo(userID int64) string {
	now := logicx.NowUTC8()
	return fmt.Sprintf("MSO%s%03d%06d", now.Format("20060102150405"), userID%1000, now.Nanosecond()/1000)
}

func RequestFromContext(ctx context.Context) (*http.Request, error) {
	req, ok := ctx.Value(utils.RequestContextKey).(*http.Request)
	if !ok || req == nil {
		return nil, errors.New("request context missing")
	}
	return req, nil
}

func ClientIPFromContext(ctx context.Context) string {
	clientIP := strings.TrimSpace(utils.GetClientIP(ctx))
	if clientIP == "" {
		return "127.0.0.1"
	}
	return clientIP
}

func BuildMemberStatusResp(user *model.User) *types.GetMemberStatusResp {
	now := logicx.NowUTC8()
	resp := &types.GetMemberStatusResp{
		Success:     true,
		IsActive:    false,
		CurrentTime: logicx.FormatUTC8Time(now),
	}
	if user == nil || user.MemberExpiresAt == nil {
		return resp
	}

	resp.MemberExpiresAt = logicx.FormatUTC8TimePtr(user.MemberExpiresAt)
	resp.IsActive = logicx.InUTC8(*user.MemberExpiresAt).After(now)
	return resp
}

func BuildMemberOrderStatusResp(order *model.MemberSubscriptionOrder) *types.GetMemberSubscriptionOrderStatusResp {
	if order == nil {
		return &types.GetMemberSubscriptionOrderStatusResp{
			Success: false,
			Message: "订单不存在",
		}
	}

	return &types.GetMemberSubscriptionOrderStatusResp{
		Success:               true,
		OrderNo:               order.OrderNo,
		PayChannel:            order.PayChannel,
		PlanCode:              order.PlanCode,
		PlanName:              order.PlanName,
		AmountFen:             order.AmountFen,
		AmountYuan:            FormatAmountFenToYuan(order.AmountFen),
		Status:                order.Status,
		PaidAt:                logicx.FormatUTC8TimePtr(order.PaidAt),
		MemberExpiresAtBefore: logicx.FormatUTC8TimePtr(order.MemberExpiresAtBefore),
		MemberExpiresAtAfter:  logicx.FormatUTC8TimePtr(order.MemberExpiresAtAfter),
	}
}

func (s *GatewayService) CreatePayPayload(ctx context.Context, order *model.MemberSubscriptionOrder) (string, *types.WechatAppPayParams, error) {
	if order == nil {
		return "", nil, errors.New("member subscription order is nil")
	}

	switch order.PayChannel {
	case PayChannelAlipay:
		orderString, err := s.createAlipayOrderString(ctx, order)
		return orderString, nil, err
	case PayChannelWechat:
		params, err := s.createWechatAppPayParams(ctx, order)
		return "", params, err
	default:
		return "", nil, fmt.Errorf("unsupported pay channel: %s", order.PayChannel)
	}
}

func (s *GatewayService) ReconcileOrder(ctx context.Context, order *model.MemberSubscriptionOrder) (*model.MemberSubscriptionOrder, error) {
	if order == nil {
		return nil, errors.New("member subscription order is nil")
	}
	if order.Status != model.MemberSubscriptionOrderStatusPending {
		return order, nil
	}

	switch order.PayChannel {
	case PayChannelAlipay:
		return s.reconcileAlipayOrder(ctx, order)
	case PayChannelWechat:
		return s.reconcileWechatOrder(ctx, order)
	default:
		return order, nil
	}
}

func (s *GatewayService) HandleAlipayNotify(ctx context.Context) (*model.MemberSubscriptionOrder, error) {
	req, err := RequestFromContext(ctx)
	if err != nil {
		return nil, err
	}

	bodyMap, err := gopayalipay.ParseNotifyToBodyMap(req)
	if err != nil {
		return nil, err
	}

	publicKey := strings.TrimSpace(s.svcCtx.Config.Alipay.PublicKey)
	if publicKey == "" {
		return nil, errors.New("alipay public key is not configured")
	}

	ok, err := gopayalipay.VerifySign(publicKey, bodyMap)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("alipay notify sign verify failed")
	}

	tradeStatus := strings.TrimSpace(bodyMap.GetString("trade_status"))
	if tradeStatus != "TRADE_SUCCESS" && tradeStatus != "TRADE_FINISHED" {
		return nil, nil
	}

	paidAmountFen, err := ParseAmountYuanToFen(bodyMap.GetString("total_amount"))
	if err != nil {
		return nil, err
	}

	paidAt, err := ParseUTC8Time(bodyMap.GetString("gmt_payment"))
	if err != nil {
		paidAt = logicx.NowUTC8()
	}

	service := NewMemberSubscriptionService(s.svcCtx, logicx.NowUTC8)
	return service.MarkOrderPaid(
		bodyMap.GetString("out_trade_no"),
		bodyMap.GetString("trade_no"),
		paidAmountFen,
		paidAt,
	)
}

func (s *GatewayService) HandleWechatNotify(ctx context.Context) (*model.MemberSubscriptionOrder, error) {
	req, err := RequestFromContext(ctx)
	if err != nil {
		return nil, err
	}

	client, err := s.newWechatClient(true)
	if err != nil {
		return nil, err
	}

	notifyReq, err := gopaywechat.V3ParseNotify(req)
	if err != nil {
		return nil, err
	}

	publicKey, ok := client.SnCertMap.Load(notifyReq.SignInfo.HeaderSerial)
	if !ok {
		return nil, fmt.Errorf("wechat pay serial not found: %s", notifyReq.SignInfo.HeaderSerial)
	}
	if err := notifyReq.VerifySignByPK(publicKey); err != nil {
		return nil, err
	}

	decryptResult, err := notifyReq.DecryptPayCipherText(s.svcCtx.Config.WechatPay.ApiV3Key)
	if err != nil {
		return nil, err
	}
	if decryptResult == nil || decryptResult.TradeState != gopaywechat.TradeStateSuccess {
		return nil, nil
	}
	if decryptResult.Amount == nil {
		return nil, errors.New("wechat pay notify amount is missing")
	}

	paidAt, err := ParseRFC3339ToUTC8(decryptResult.SuccessTime)
	if err != nil {
		paidAt = logicx.NowUTC8()
	}

	service := NewMemberSubscriptionService(s.svcCtx, logicx.NowUTC8)
	return service.MarkOrderPaid(
		decryptResult.OutTradeNo,
		decryptResult.TransactionId,
		decryptResult.Amount.Total,
		paidAt,
	)
}

func (s *GatewayService) createAlipayOrderString(ctx context.Context, order *model.MemberSubscriptionOrder) (string, error) {
	client, err := s.newAlipayClient()
	if err != nil {
		return "", err
	}

	bm := make(gopay.BodyMap)
	bm.Set("subject", order.PlanName)
	bm.Set("out_trade_no", order.OrderNo)
	bm.Set("total_amount", FormatAmountFenToYuan(order.AmountFen))
	bm.Set("product_code", "QUICK_MSECURITY_PAY")
	bm.Set("body", fmt.Sprintf("%s-%s", order.PlanName, order.OrderNo))

	return client.TradeAppPay(ctx, bm)
}

func (s *GatewayService) createWechatAppPayParams(ctx context.Context, order *model.MemberSubscriptionOrder) (*types.WechatAppPayParams, error) {
	client, err := s.newWechatClient(false)
	if err != nil {
		return nil, err
	}
	appID := strconv.FormatInt(s.svcCtx.Config.WechatPay.AppId, 10)
	mchID := strconv.FormatInt(s.svcCtx.Config.WechatPay.MchId, 10)

	bm := make(gopay.BodyMap)
	bm.Set("appid", appID)
	bm.Set("mchid", mchID)
	bm.Set("description", order.PlanName)
	bm.Set("out_trade_no", order.OrderNo)
	bm.Set("notify_url", s.svcCtx.Config.WechatPay.NotifyUrl)
	bm.SetBodyMap("amount", func(b gopay.BodyMap) {
		b.Set("total", order.AmountFen)
		b.Set("currency", "CNY")
	})
	bm.SetBodyMap("scene_info", func(b gopay.BodyMap) {
		b.Set("payer_client_ip", ClientIPFromContext(ctx))
	})

	wxRsp, err := client.V3TransactionApp(ctx, bm)
	if err != nil {
		return nil, err
	}
	if wxRsp == nil || wxRsp.Response == nil {
		return nil, errors.New("wechat app pay response is empty")
	}
	if wxRsp.Code != 0 {
		return nil, fmt.Errorf("wechat app pay create failed: %s", wxRsp.Error)
	}

	appParams, err := client.PaySignOfApp(appID, wxRsp.Response.PrepayId)
	if err != nil {
		return nil, err
	}

	return &types.WechatAppPayParams{
		Appid:     appParams.Appid,
		Partnerid: appParams.Partnerid,
		Prepayid:  appParams.Prepayid,
		Package:   appParams.Package,
		Noncestr:  appParams.Noncestr,
		Timestamp: appParams.Timestamp,
		Sign:      appParams.Sign,
	}, nil
}

func (s *GatewayService) reconcileAlipayOrder(ctx context.Context, order *model.MemberSubscriptionOrder) (*model.MemberSubscriptionOrder, error) {
	client, err := s.newAlipayClient()
	if err != nil {
		return order, err
	}

	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", order.OrderNo)
	queryResp, err := client.TradeQuery(ctx, bm)
	if err != nil {
		return order, err
	}
	if queryResp == nil || queryResp.Response == nil {
		return order, nil
	}

	switch strings.TrimSpace(queryResp.Response.TradeStatus) {
	case "TRADE_SUCCESS", "TRADE_FINISHED":
		paidAmountFen, err := ParseAmountYuanToFen(queryResp.Response.TotalAmount)
		if err != nil {
			return order, err
		}
		paidAt, err := ParseUTC8Time(queryResp.Response.SendPayDate)
		if err != nil {
			paidAt = logicx.NowUTC8()
		}
		service := NewMemberSubscriptionService(s.svcCtx, logicx.NowUTC8)
		return service.MarkOrderPaid(order.OrderNo, queryResp.Response.TradeNo, paidAmountFen, paidAt)
	case "TRADE_CLOSED":
		order.Status = model.MemberSubscriptionOrderStatusClosed
		if err := s.svcCtx.MemberSubscriptionOrderModel.UpdateWithTx(nil, order); err != nil {
			return order, err
		}
	}

	return order, nil
}

func (s *GatewayService) reconcileWechatOrder(ctx context.Context, order *model.MemberSubscriptionOrder) (*model.MemberSubscriptionOrder, error) {
	client, err := s.newWechatClient(false)
	if err != nil {
		return order, err
	}

	queryResp, err := client.V3TransactionQueryOrder(ctx, gopaywechat.OutTradeNo, order.OrderNo)
	if err != nil {
		return order, err
	}
	if queryResp == nil || queryResp.Response == nil {
		return order, nil
	}

	switch queryResp.Response.TradeState {
	case gopaywechat.TradeStateSuccess:
		if queryResp.Response.Amount == nil {
			return order, errors.New("wechat pay query amount is missing")
		}
		paidAt, err := ParseRFC3339ToUTC8(queryResp.Response.SuccessTime)
		if err != nil {
			paidAt = logicx.NowUTC8()
		}
		service := NewMemberSubscriptionService(s.svcCtx, logicx.NowUTC8)
		return service.MarkOrderPaid(order.OrderNo, queryResp.Response.TransactionId, queryResp.Response.Amount.Total, paidAt)
	case gopaywechat.TradeStateClosed, gopaywechat.TradeStateRevoked, gopaywechat.TradeStatePayError:
		order.Status = model.MemberSubscriptionOrderStatusClosed
		if err := s.svcCtx.MemberSubscriptionOrderModel.UpdateWithTx(nil, order); err != nil {
			return order, err
		}
	}

	return order, nil
}

func (s *GatewayService) newAlipayClient() (*gopayalipay.Client, error) {
	cfg := s.svcCtx.Config.Alipay
	appID := strconv.FormatInt(cfg.AppId, 10)
	if appID == "0" || strings.TrimSpace(cfg.PrivateKey) == "" {
		return nil, errors.New("alipay config is incomplete")
	}

	client, err := gopayalipay.NewClient(appID, cfg.PrivateKey, config.IsProductionEnv(s.svcCtx.Config.AppEnv))
	if err != nil {
		return nil, err
	}

	signType := strings.TrimSpace(cfg.SignType)
	if signType == "" {
		signType = gopayalipay.RSA2
	}
	charset := strings.TrimSpace(cfg.Charset)
	if charset == "" {
		charset = "utf-8"
	}

	client.SetCharset(charset).
		SetSignType(signType).
		SetNotifyUrl(strings.TrimSpace(cfg.NotifyUrl)).
		SetReturnUrl(strings.TrimSpace(cfg.ReturnUrl))

	return client, nil
}

func (s *GatewayService) newWechatClient(enableVerify bool) (*gopaywechat.ClientV3, error) {
	cfg := s.svcCtx.Config.WechatPay
	appID := strconv.FormatInt(cfg.AppId, 10)
	mchID := strconv.FormatInt(cfg.MchId, 10)
	if appID == "0" || mchID == "0" || strings.TrimSpace(cfg.SerialNo) == "" ||
		strings.TrimSpace(cfg.ApiV3Key) == "" || strings.TrimSpace(cfg.PrivateKey) == "" {
		return nil, errors.New("wechat pay config is incomplete")
	}

	client, err := gopaywechat.NewClientV3(mchID, cfg.SerialNo, cfg.ApiV3Key, cfg.PrivateKey)
	if err != nil {
		return nil, err
	}
	if enableVerify {
		if err := client.AutoVerifySign(false); err != nil {
			return nil, err
		}
	}
	return client, nil
}
