package sms

import (
	"context"
	"testing"

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
