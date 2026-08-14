package eventnews

import (
	"context"
	"testing"
	"time"
)

func TestEventNewsCacheMetricsTrackCacheOutcomes(t *testing.T) {
	ResetEventNewsCacheMetricsForTest()
	t.Cleanup(ResetEventNewsCacheMetricsForTest)

	svcCtx := newEventNewsTestSvc(t)
	attachEventNewsTestRedis(t, svcCtx)
	ctx := context.Background()

	storeCachedJSON(ctx, svcCtx, "eventnews:test:hit", map[string]string{"value": "cached"}, time.Minute)
	var hit map[string]string
	if !loadCachedJSON(ctx, svcCtx, "eventnews:test:hit", &hit) || hit["value"] != "cached" {
		t.Fatalf("expected cache hit, got %#v", hit)
	}
	if loadCachedJSON(ctx, svcCtx, "eventnews:test:miss", &map[string]string{}) {
		t.Fatal("expected cache miss")
	}
	if err := svcCtx.Redis.Set(ctx, "eventnews:test:broken", "{", time.Minute).Err(); err != nil {
		t.Fatalf("store broken payload: %v", err)
	}
	if loadCachedJSON(ctx, svcCtx, "eventnews:test:broken", &map[string]string{}) {
		t.Fatal("expected broken payload fallback")
	}
	eventNewsCacheMetrics.Fallback()

	if err := svcCtx.Redis.Close(); err != nil {
		t.Fatalf("close redis: %v", err)
	}
	storeCachedJSON(ctx, svcCtx, "eventnews:test:write-error", map[string]string{"value": "x"}, time.Minute)

	snapshot := EventNewsCacheMetrics()
	if snapshot.Hits != 1 || snapshot.Misses != 1 || snapshot.DecodeErrors != 1 || snapshot.Fallbacks != 1 {
		t.Fatalf("unexpected cache counters: %+v", snapshot)
	}
	if snapshot.WriteErrors != 1 {
		t.Fatalf("write errors = %d, want 1", snapshot.WriteErrors)
	}
}
