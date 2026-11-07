package staticread

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/svc"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestLoadTTLCanBeInvalidated(t *testing.T) {
	ResetCacheForTest()
	miniRedis := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	svcCtx := &svc.ServiceContext{Redis: client}
	loads := 0
	load := func() (string, error) {
		loads++
		return "value", nil
	}
	if _, err := LoadTTL(context.Background(), svcCtx, "runtime-config:test", time.Minute, load); err != nil {
		t.Fatalf("load cold value: %v", err)
	}
	if _, err := LoadTTL(context.Background(), svcCtx, "runtime-config:test", time.Minute, load); err != nil {
		t.Fatalf("load cached value: %v", err)
	}
	if loads != 1 {
		t.Fatalf("authoritative loads before invalidation = %d, want 1", loads)
	}
	Invalidate(context.Background(), svcCtx, "runtime-config:test")
	if _, err := LoadTTL(context.Background(), svcCtx, "runtime-config:test", time.Minute, load); err != nil {
		t.Fatalf("load invalidated value: %v", err)
	}
	if loads != 2 {
		t.Fatalf("authoritative loads after invalidation = %d, want 2", loads)
	}
}
