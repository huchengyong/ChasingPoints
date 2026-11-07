package venue

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"chasing_points/internal/observability"
	"chasing_points/internal/readcache"
	"chasing_points/internal/svc"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

const (
	venueListCacheTTL   = 2 * time.Minute
	venueDetailCacheTTL = 2 * time.Minute
	venueNearbyCacheTTL = time.Minute
)

var (
	venueCacheMetrics observability.CacheMetrics
	venueCacheGroup   singleflight.Group
)

func loadVenueCache[T any](ctx context.Context, svcCtx *svc.ServiceContext, scope string, params interface{}, ttl time.Duration, load func() (T, error)) (T, error) {
	version := venueCacheVersion(ctx, svcCtx)
	key := fmt.Sprintf("venue:%s:v%d:%s", scope, version, venueCacheHash(params))
	var client *redis.Client
	if svcCtx != nil {
		client = svcCtx.Redis
	}
	return readcache.LoadJSON(ctx, client, key, ttl, &venueCacheMetrics, &venueCacheGroup, load)
}

func BumpVenueCacheVersion(ctx context.Context, svcCtx *svc.ServiceContext) error {
	if svcCtx == nil || svcCtx.Redis == nil {
		return nil
	}
	if err := svcCtx.Redis.Incr(ctx, "venue:version").Err(); err != nil {
		venueCacheMetrics.WriteError(ctx)
		return err
	}
	return nil
}

func VenueCacheMetrics() observability.CacheSnapshot {
	return venueCacheMetrics.Snapshot()
}

func ResetVenueCacheForTest() {
	venueCacheMetrics.Reset()
	venueCacheGroup = singleflight.Group{}
}

func venueCacheVersion(ctx context.Context, svcCtx *svc.ServiceContext) int64 {
	if svcCtx == nil || svcCtx.Redis == nil {
		return 0
	}
	version, err := svcCtx.Redis.Get(ctx, "venue:version").Int64()
	if err == nil {
		return version
	}
	if err != redis.Nil {
		venueCacheMetrics.RedisError(ctx)
	}
	return 0
}

func venueCacheHash(params interface{}) string {
	payload, err := json.Marshal(params)
	if err != nil {
		return "fallback"
	}
	sum := sha1.Sum(payload)
	return hex.EncodeToString(sum[:])
}

func venueLocationBucket(value float64) float64 {
	return math.Round(value*1000) / 1000
}
