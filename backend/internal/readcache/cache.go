package readcache

import (
	"context"
	"encoding/json"
	"time"

	"chasing_points/internal/observability"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/sync/singleflight"
)

func LoadJSON[T any](
	ctx context.Context,
	client *redis.Client,
	key string,
	ttl time.Duration,
	metrics *observability.CacheMetrics,
	group *singleflight.Group,
	load func() (T, error),
) (T, error) {
	if client != nil {
		if cached, ok := readJSON[T](ctx, client, key, metrics, true); ok {
			return cached, nil
		}
	}
	loadOnce := func() (interface{}, error) {
		if client != nil {
			if cached, ok := readJSON[T](ctx, client, key, metrics, false); ok {
				return cached, nil
			}
		}
		if metrics != nil {
			metrics.Fallback(ctx)
		}
		value, err := load()
		if err != nil {
			return nil, err
		}
		storeJSON(ctx, client, key, ttl, value, metrics)
		return value, nil
	}
	if group == nil {
		value, err := loadOnce()
		if err != nil {
			var zero T
			return zero, err
		}
		return value.(T), nil
	}
	value, err, _ := group.Do(key, loadOnce)
	if err != nil {
		var zero T
		return zero, err
	}
	return value.(T), nil
}

func readJSON[T any](ctx context.Context, client *redis.Client, key string, metrics *observability.CacheMetrics, record bool) (T, bool) {
	var value T
	payload, err := client.Get(ctx, key).Bytes()
	if err == nil {
		if err := json.Unmarshal(payload, &value); err == nil {
			if record && metrics != nil {
				metrics.Hit(ctx)
			}
			return value, true
		}
		if metrics != nil {
			metrics.DecodeError(ctx)
		}
		if delErr := client.Del(ctx, key).Err(); delErr != nil && metrics != nil {
			metrics.WriteError(ctx)
		}
		return value, false
	}
	if err == redis.Nil {
		if record && metrics != nil {
			metrics.Miss(ctx)
		}
		return value, false
	}
	if metrics != nil {
		metrics.RedisError(ctx)
	}
	logx.WithContext(ctx).Errorf("读取共享缓存失败: key=%s err=%v", key, err)
	return value, false
}

func storeJSON[T any](ctx context.Context, client *redis.Client, key string, ttl time.Duration, value T, metrics *observability.CacheMetrics) {
	if client == nil {
		return
	}
	payload, err := json.Marshal(value)
	if err != nil {
		if metrics != nil {
			metrics.WriteError(ctx)
		}
		return
	}
	if err := client.Set(ctx, key, payload, ttl).Err(); err != nil {
		if metrics != nil {
			metrics.WriteError(ctx)
		}
		logx.WithContext(ctx).Errorf("写入共享缓存失败: key=%s err=%v", key, err)
	}
}
