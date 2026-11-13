package sms

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newCodeManagerForTest(t *testing.T) (*CodeManager, *miniredis.Miniredis) {
	t.Helper()
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() {
		_ = redisClient.Close()
	})
	return NewCodeManager(redisClient), redisServer
}

func TestDeleteAccountCodeAcceptsValidCodeOnce(t *testing.T) {
	manager, _ := newCodeManagerForTest(t)
	ctx := context.Background()

	if err := manager.SaveCodeForScene(ctx, "13800138000", SceneDeleteAccount, "123456"); err != nil {
		t.Fatalf("save delete-account code: %v", err)
	}

	valid, err := manager.VerifyCodeForScene(ctx, "13800138000", SceneDeleteAccount, "123456")
	if err != nil || !valid {
		t.Fatalf("valid delete-account code = (%v, %v), want (true, nil)", valid, err)
	}

	valid, err = manager.VerifyCodeForScene(ctx, "13800138000", SceneDeleteAccount, "123456")
	if err == nil || valid {
		t.Fatalf("replayed delete-account code = (%v, %v), want (false, error)", valid, err)
	}
}

func TestDeleteAccountCodeRejectsWrongCodeAndKeepsOriginal(t *testing.T) {
	manager, _ := newCodeManagerForTest(t)
	ctx := context.Background()

	if err := manager.SaveCodeForScene(ctx, "13800138001", SceneDeleteAccount, "123456"); err != nil {
		t.Fatalf("save delete-account code: %v", err)
	}

	valid, err := manager.VerifyCodeForScene(ctx, "13800138001", SceneDeleteAccount, "654321")
	if err != nil || valid {
		t.Fatalf("wrong delete-account code = (%v, %v), want (false, nil)", valid, err)
	}

	valid, err = manager.VerifyCodeForScene(ctx, "13800138001", SceneDeleteAccount, "123456")
	if err != nil || !valid {
		t.Fatalf("original code after wrong attempt = (%v, %v), want (true, nil)", valid, err)
	}
}

func TestDeleteAccountCodeExpires(t *testing.T) {
	manager, redisServer := newCodeManagerForTest(t)
	ctx := context.Background()

	if err := manager.SaveCodeForScene(ctx, "13800138002", SceneDeleteAccount, "123456"); err != nil {
		t.Fatalf("save delete-account code: %v", err)
	}
	redisServer.FastForward(CodeExpiration)

	valid, err := manager.VerifyCodeForScene(ctx, "13800138002", SceneDeleteAccount, "123456")
	if err == nil || valid {
		t.Fatalf("expired delete-account code = (%v, %v), want (false, error)", valid, err)
	}
}

func TestCodesAreIsolatedByScene(t *testing.T) {
	manager, _ := newCodeManagerForTest(t)
	ctx := context.Background()

	if err := manager.SaveCodeForScene(ctx, "13800138003", SceneLogin, "111111"); err != nil {
		t.Fatalf("save login code: %v", err)
	}
	if err := manager.SaveCodeForScene(ctx, "13800138003", SceneDeleteAccount, "222222"); err != nil {
		t.Fatalf("save delete-account code: %v", err)
	}

	valid, err := manager.VerifyCodeForScene(ctx, "13800138003", SceneDeleteAccount, "111111")
	if err != nil || valid {
		t.Fatalf("login code accepted for delete-account = (%v, %v), want (false, nil)", valid, err)
	}
	valid, err = manager.VerifyCodeForScene(ctx, "13800138003", SceneLogin, "111111")
	if err != nil || !valid {
		t.Fatalf("login code after scene mismatch = (%v, %v), want (true, nil)", valid, err)
	}
}

func TestCodeIsVoidedAfterMaxWrongAttempts(t *testing.T) {
	manager, _ := newCodeManagerForTest(t)
	ctx := context.Background()
	phone := "13800138004"

	if err := manager.SaveCode(ctx, phone, "123456"); err != nil {
		t.Fatalf("save code: %v", err)
	}
	for i := 1; i <= MaxVerifyAttempts; i++ {
		valid, err := manager.VerifyCode(ctx, phone, "000000")
		if err != nil || valid {
			t.Fatalf("wrong attempt %d = (%v, %v), want (false, nil)", i, valid, err)
		}
	}

	valid, err := manager.VerifyCode(ctx, phone, "123456")
	if !errors.Is(err, ErrCodeExpired) || valid {
		t.Fatalf("correct code after %d wrong attempts = (%v, %v), want (false, ErrCodeExpired)", MaxVerifyAttempts, valid, err)
	}
}

func TestNewCodeResetsWrongAttempts(t *testing.T) {
	manager, _ := newCodeManagerForTest(t)
	ctx := context.Background()
	phone := "13800138005"

	if err := manager.SaveCode(ctx, phone, "123456"); err != nil {
		t.Fatalf("save code: %v", err)
	}
	for i := 1; i < MaxVerifyAttempts; i++ {
		if valid, err := manager.VerifyCode(ctx, phone, "000000"); err != nil || valid {
			t.Fatalf("wrong attempt %d = (%v, %v), want (false, nil)", i, valid, err)
		}
	}
	if err := manager.SaveCode(ctx, phone, "654321"); err != nil {
		t.Fatalf("save new code: %v", err)
	}
	// 新验证码仍有完整的错误次数：再错 MaxVerifyAttempts-1 次后正确验证码依然有效。
	for i := 1; i < MaxVerifyAttempts; i++ {
		if valid, err := manager.VerifyCode(ctx, phone, "000000"); err != nil || valid {
			t.Fatalf("wrong attempt %d on new code = (%v, %v), want (false, nil)", i, valid, err)
		}
	}
	if valid, err := manager.VerifyCode(ctx, phone, "654321"); err != nil || !valid {
		t.Fatalf("new code = (%v, %v), want (true, nil)", valid, err)
	}
}

func TestWrongAttemptsAreIsolatedByScene(t *testing.T) {
	manager, _ := newCodeManagerForTest(t)
	ctx := context.Background()
	phone := "13800138006"

	if err := manager.SaveCodeForScene(ctx, phone, SceneLogin, "111111"); err != nil {
		t.Fatalf("save login code: %v", err)
	}
	if err := manager.SaveCodeForScene(ctx, phone, SceneBind, "222222"); err != nil {
		t.Fatalf("save bind code: %v", err)
	}
	for i := 1; i <= MaxVerifyAttempts; i++ {
		if _, err := manager.VerifyCodeForScene(ctx, phone, SceneBind, "000000"); err != nil {
			t.Fatalf("wrong bind attempt %d: %v", i, err)
		}
	}
	if valid, err := manager.VerifyCodeForScene(ctx, phone, SceneLogin, "111111"); err != nil || !valid {
		t.Fatalf("login code after bind code voided = (%v, %v), want (true, nil)", valid, err)
	}
}

func TestSendQuotaLimitsPhonePerDay(t *testing.T) {
	manager, redisServer := newCodeManagerForTest(t)
	ctx := context.Background()
	phone := "13800138007"

	for i := 1; i <= PhoneDailySendLimit; i++ {
		if err := manager.ReserveSendQuota(ctx, phone, ""); err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
	}
	if err := manager.ReserveSendQuota(ctx, phone, ""); !errors.Is(err, ErrSendLimited) {
		t.Fatalf("send over phone limit = %v, want ErrSendLimited", err)
	}
	if err := manager.ReserveSendQuota(ctx, "13800138008", ""); err != nil {
		t.Fatalf("other phone: %v", err)
	}

	redisServer.FastForward(24 * time.Hour)
	if err := manager.ReserveSendQuota(ctx, phone, ""); err != nil {
		t.Fatalf("send after window: %v", err)
	}
}

func TestSendQuotaLimitsIPPerHourAcrossPhones(t *testing.T) {
	manager, redisServer := newCodeManagerForTest(t)
	ctx := context.Background()
	ip := "203.0.113.7"

	for i := 0; i < IPHourlySendLimit; i++ {
		phone := fmt.Sprintf("139%08d", i)
		if err := manager.ReserveSendQuota(ctx, phone, ip); err != nil {
			t.Fatalf("send %d: %v", i+1, err)
		}
	}
	if err := manager.ReserveSendQuota(ctx, "13700000000", ip); !errors.Is(err, ErrSendLimited) {
		t.Fatalf("send over ip limit = %v, want ErrSendLimited", err)
	}
	if err := manager.ReserveSendQuota(ctx, "13700000001", "203.0.113.8"); err != nil {
		t.Fatalf("other ip: %v", err)
	}

	redisServer.FastForward(time.Hour)
	if err := manager.ReserveSendQuota(ctx, "13700000002", ip); err != nil {
		t.Fatalf("send after window: %v", err)
	}
}
