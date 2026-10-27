package public

import (
	"context"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"chasing_points/internal/svc"
)

func TestPreviewRateLimitAllowsBurstThenBlocks(t *testing.T) {
	mr := miniredis.RunT(t)
	svcCtx := &svc.ServiceContext{Redis: redis.NewClient(&redis.Options{Addr: mr.Addr()})}
	logic := NewPreviewPublicMatchInviteLogic(context.Background(), svcCtx)

	for i := 0; i < previewRateLimitPerMinute; i++ {
		allowed, err := logic.allowRequest()
		if err != nil {
			t.Fatalf("request %d: %v", i+1, err)
		}
		if !allowed {
			t.Fatalf("request %d must be allowed", i+1)
		}
	}

	allowed, err := logic.allowRequest()
	if err != nil {
		t.Fatalf("request over the limit: %v", err)
	}
	if allowed {
		t.Fatalf("request over the limit must be blocked")
	}
	if !mr.Exists(previewRateLimitKeyPrefix + "unknown") {
		t.Fatalf("rate limit key must be set")
	}
}

func TestPreviewRateLimitFailsClosedWithoutRedis(t *testing.T) {
	logic := NewPreviewPublicMatchInviteLogic(context.Background(), &svc.ServiceContext{})

	allowed, err := logic.allowRequest()
	if err == nil {
		t.Fatalf("missing redis must return an error")
	}
	if allowed {
		t.Fatalf("missing redis must not allow the request")
	}
}
