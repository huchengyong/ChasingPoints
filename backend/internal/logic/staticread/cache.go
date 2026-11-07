package staticread

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"chasing_points/internal/observability"
	"chasing_points/internal/readcache"
	"chasing_points/internal/svc"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

const staticReadCacheTTL = 24 * time.Hour

var (
	staticReadCacheMetrics observability.CacheMetrics
	staticReadCacheGroup   singleflight.Group
)

func Load[T any](ctx context.Context, svcCtx *svc.ServiceContext, key string, load func() (T, error)) (T, error) {
	return LoadTTL(ctx, svcCtx, key, staticReadCacheTTL, load)
}

func LoadTTL[T any](ctx context.Context, svcCtx *svc.ServiceContext, key string, ttl time.Duration, load func() (T, error)) (T, error) {
	var client *redis.Client
	if svcCtx != nil {
		client = svcCtx.Redis
	}
	return readcache.LoadJSON(ctx, client, cacheKey(key), ttl, &staticReadCacheMetrics, &staticReadCacheGroup, load)
}

func Invalidate(ctx context.Context, svcCtx *svc.ServiceContext, key string) {
	if svcCtx == nil || svcCtx.Redis == nil {
		return
	}
	if err := svcCtx.Redis.Del(ctx, cacheKey(key)).Err(); err != nil {
		staticReadCacheMetrics.WriteError(ctx)
	}
}

func cacheKey(key string) string {
	return "static-read:v1:" + strings.TrimSpace(key)
}

func Key(scope string, params ...interface{}) string {
	payload, err := json.Marshal(params)
	if err != nil {
		return scope
	}
	sum := sha1.Sum(payload)
	return strings.TrimSpace(scope) + ":" + hex.EncodeToString(sum[:])
}

func CacheMetrics() observability.CacheSnapshot {
	return staticReadCacheMetrics.Snapshot()
}

func ResetCacheForTest() {
	staticReadCacheMetrics.Reset()
	staticReadCacheGroup = singleflight.Group{}
}
