package auth

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"

	"chasing_points/internal/sms"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"
)

// 夹具没有短信客户端：超限请求必须在生成验证码和调用短信服务之前被拒绝。

func TestSendSmsRejectsPhoneOverDailyLimitBeforeSending(t *testing.T) {
	svcCtx, mr := newAuthRewardTestSvc(t)
	ctx := context.Background()
	phone := "13800138100"

	for i := 0; i < sms.PhoneDailySendLimit; i++ {
		if err := svcCtx.CodeManager.ReserveSendQuota(ctx, phone, ""); err != nil {
			t.Fatalf("reserve %d: %v", i+1, err)
		}
	}

	resp, err := NewSendSmsLogic(ctx, svcCtx).SendSms(&types.SendSmsReq{Phone: phone})
	if err != nil {
		t.Fatalf("SendSms error: %v", err)
	}
	if resp.Success || resp.Message != sms.ErrSendLimited.Error() {
		t.Fatalf("resp = %+v, want limited failure", resp)
	}
	if mr.Exists(sms.CodeKeyPrefix + sms.SceneLogin + ":" + phone) {
		t.Fatal("verification code was saved for a limited request")
	}
}

func TestSendSmsRejectsIPOverHourlyLimitBeforeSending(t *testing.T) {
	svcCtx, mr := newAuthRewardTestSvc(t)
	ip := "203.0.113.9"

	for i := 0; i < sms.IPHourlySendLimit; i++ {
		if err := svcCtx.CodeManager.ReserveSendQuota(context.Background(), fmt.Sprintf("139%08d", i), ip); err != nil {
			t.Fatalf("reserve %d: %v", i+1, err)
		}
	}

	req := httptest.NewRequest("POST", "/api/auth/send-sms", nil)
	req.Header.Set("X-Forwarded-For", ip)
	ctx := utils.WithRequest(req.Context(), req)
	phone := "13800138101"

	resp, err := NewSendSmsLogic(ctx, svcCtx).SendSms(&types.SendSmsReq{Phone: phone})
	if err != nil {
		t.Fatalf("SendSms error: %v", err)
	}
	if resp.Success || resp.Message != sms.ErrSendLimited.Error() {
		t.Fatalf("resp = %+v, want limited failure", resp)
	}
	if mr.Exists(sms.CodeKeyPrefix + sms.SceneLogin + ":" + phone) {
		t.Fatal("verification code was saved for a limited request")
	}
}
