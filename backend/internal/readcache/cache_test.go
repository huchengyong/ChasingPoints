package readcache

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"chasing_points/internal/observability"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

type cacheTestPayload struct {
	Value string `json:"value"`
}

func TestLoadJSONFallsBackFromCorruptPayloadAndRepairsCache(t *testing.T) {
	miniRedis := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	if err := client.Set(context.Background(), "corrupt", "{bad-json", time.Minute).Err(); err != nil {
		t.Fatalf("seed corrupt cache: %v", err)
	}
	var metrics observability.CacheMetrics
	var group singleflight.Group
	loads := 0
	load := func() (cacheTestPayload, error) {
		loads++
		return cacheTestPayload{Value: "database"}, nil
	}
	first, err := LoadJSON(context.Background(), client, "corrupt", time.Minute, &metrics, &group, load)
	second, secondErr := LoadJSON(context.Background(), client, "corrupt", time.Minute, &metrics, &group, load)
	if err != nil || secondErr != nil || first.Value != "database" || second.Value != "database" || loads != 1 {
		t.Fatalf("corrupt cache fallback failed: first=%+v second=%+v loads=%d err=%v secondErr=%v", first, second, loads, err, secondErr)
	}
	snapshot := metrics.Snapshot()
	if snapshot.DecodeErrors != 1 || snapshot.Fallbacks != 1 || snapshot.Hits != 1 {
		t.Fatalf("unexpected corrupt cache metrics: %+v", snapshot)
	}
}

func TestLoadJSONRecordsOutcomesOnRequestMetrics(t *testing.T) {
	miniRedis := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	var metrics observability.CacheMetrics
	requestMetrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.Background(), requestMetrics)
	loads := 0
	load := func() (cacheTestPayload, error) {
		loads++
		return cacheTestPayload{Value: "database"}, nil
	}
	if _, err := LoadJSON(ctx, client, "request-metrics", time.Minute, &metrics, nil, load); err != nil {
		t.Fatalf("cold cache load: %v", err)
	}
	if _, err := LoadJSON(ctx, client, "request-metrics", time.Minute, &metrics, nil, load); err != nil {
		t.Fatalf("hot cache load: %v", err)
	}
	if loads != 1 {
		t.Fatalf("authoritative loads = %d, want 1", loads)
	}
	snapshot := requestMetrics.Snapshot()
	if snapshot.CacheMisses != 1 || snapshot.CacheFallbacks != 1 || snapshot.CacheHits != 1 {
		t.Fatalf("unexpected request cache metrics: %+v", snapshot)
	}
}

func TestLoadJSONReturnsAuthoritativeValueWhenRedisIsUnavailable(t *testing.T) {
	miniRedis := miniredis.RunT(t)
	address := miniRedis.Addr()
	miniRedis.Close()
	client := redis.NewClient(&redis.Options{Addr: address, MaxRetries: 0, DialTimeout: 20 * time.Millisecond, ReadTimeout: 20 * time.Millisecond, WriteTimeout: 20 * time.Millisecond})
	t.Cleanup(func() { _ = client.Close() })
	var metrics observability.CacheMetrics
	value, err := LoadJSON(context.Background(), client, "redis-down", time.Minute, &metrics, nil, func() (cacheTestPayload, error) {
		return cacheTestPayload{Value: "database"}, nil
	})
	if err != nil || value.Value != "database" {
		t.Fatalf("Redis outage must fall back: value=%+v err=%v", value, err)
	}
	snapshot := metrics.Snapshot()
	if snapshot.RedisErrors == 0 || snapshot.WriteErrors == 0 || snapshot.Fallbacks != 1 {
		t.Fatalf("Redis outage must be observable: %+v", snapshot)
	}
}

func TestLoadJSONSingleflightCollapsesConcurrentColdMisses(t *testing.T) {
	var metrics observability.CacheMetrics
	var group singleflight.Group
	var loads atomic.Int64
	const callers = 20
	start := make(chan struct{})
	results := make(chan cacheTestPayload, callers)
	var wait sync.WaitGroup
	for index := 0; index < callers; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			value, err := LoadJSON(context.Background(), nil, "hot-key", time.Minute, &metrics, &group, func() (cacheTestPayload, error) {
				loads.Add(1)
				time.Sleep(20 * time.Millisecond)
				return cacheTestPayload{Value: "shared"}, nil
			})
			if err != nil {
				t.Errorf("load hot key: %v", err)
				return
			}
			results <- value
		}()
	}
	close(start)
	wait.Wait()
	close(results)
	if loads.Load() != 1 {
		t.Fatalf("expected one authoritative load, got %d", loads.Load())
	}
	for result := range results {
		if result.Value != "shared" {
			t.Fatalf("unexpected shared result: %+v", result)
		}
	}
}
