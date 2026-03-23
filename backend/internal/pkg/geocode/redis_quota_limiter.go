package geocode

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var reserveQuotaScript = redis.NewScript(`
local current = redis.call("GET", KEYS[1])
if current and tonumber(current) >= tonumber(ARGV[1]) then
	return 0
end
local next = redis.call("INCR", KEYS[1])
if next == 1 then
	redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
if next > tonumber(ARGV[1]) then
	return 0
end
return 1
`)

type RedisQuotaLimiter struct {
	client    *redis.Client
	keyPrefix string
	ttl       time.Duration
}

func NewRedisQuotaLimiter(client *redis.Client, keyPrefix string, ttl time.Duration) *RedisQuotaLimiter {
	if keyPrefix == "" {
		keyPrefix = "geo:quota"
	}
	if ttl <= 0 {
		ttl = 70 * time.Second
	}

	return &RedisQuotaLimiter{
		client:    client,
		keyPrefix: keyPrefix,
		ttl:       ttl,
	}
}

func (l *RedisQuotaLimiter) CurrentUsage(ctx context.Context, accountID int64) (int64, error) {
	if l.client == nil {
		return 0, nil
	}

	key := l.key(accountID, time.Now())
	value, err := l.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return value, err
}

func (l *RedisQuotaLimiter) TryReserve(ctx context.Context, accountID int64, limit int) (bool, error) {
	if l.client == nil {
		return true, nil
	}
	if limit <= 0 {
		limit = 10
	}

	key := l.key(accountID, time.Now())
	result, err := reserveQuotaScript.Run(ctx, l.client, []string{key}, limit, l.ttl.Milliseconds()).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func (l *RedisQuotaLimiter) key(accountID int64, now time.Time) string {
	return fmt.Sprintf("%s:%d:%s", l.keyPrefix, accountID, now.Format("200601021504"))
}
