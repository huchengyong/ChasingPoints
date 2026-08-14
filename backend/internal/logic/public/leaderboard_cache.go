package public

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"chasing_points/internal/observability"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/sync/singleflight"
)

const leaderboardCacheTTL = 20 * time.Second

var (
	leaderboardCacheMetrics observability.CacheMetrics
	leaderboardCacheGroup   singleflight.Group
)

type leaderboardSharedPayload struct {
	Total         int64                   `json:"total"`
	TopThree      []types.LeaderboardItem `json:"top_three"`
	List          []types.LeaderboardItem `json:"list"`
	RankNames     map[int]string          `json:"rank_names"`
	ConfigVersion string                  `json:"config_version"`
}

func LeaderboardCacheMetrics() observability.CacheSnapshot {
	return leaderboardCacheMetrics.Snapshot()
}

func ResetLeaderboardCacheForTest() {
	leaderboardCacheMetrics.Reset()
	leaderboardCacheGroup = singleflight.Group{}
}

func BumpLeaderboardCacheVersion(ctx context.Context, svcCtx *svc.ServiceContext, gameType int) error {
	if svcCtx == nil || svcCtx.Redis == nil {
		return nil
	}
	if err := svcCtx.Redis.Incr(ctx, leaderboardVersionKey(gameType)).Err(); err != nil {
		leaderboardCacheMetrics.WriteError(ctx)
		return err
	}
	return nil
}

func BumpAllLeaderboardCacheVersions(ctx context.Context, svcCtx *svc.ServiceContext) error {
	var firstErr error
	for gameType := 1; gameType <= 4; gameType++ {
		if err := BumpLeaderboardCacheVersion(ctx, svcCtx, gameType); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func loadLeaderboardShared(
	ctx context.Context,
	svcCtx *svc.ServiceContext,
	kind string,
	gameType, page, pageSize int,
	load func() (*leaderboardSharedPayload, error),
) (*leaderboardSharedPayload, error) {
	version, redisReady := leaderboardCacheVersion(ctx, svcCtx, gameType)
	key := fmt.Sprintf("leaderboard:%s:v%d:game:%d:page:%d:size:%d", kind, version, gameType, page, pageSize)
	if redisReady {
		if cached, ok := readLeaderboardCache(ctx, svcCtx, key, true); ok {
			return cached, nil
		}
	}
	value, err, _ := leaderboardCacheGroup.Do(key, func() (interface{}, error) {
		if redisReady {
			if cached, ok := readLeaderboardCache(ctx, svcCtx, key, false); ok {
				return cached, nil
			}
		}
		leaderboardCacheMetrics.Fallback(ctx)
		loaded, loadErr := load()
		if loadErr != nil {
			return nil, loadErr
		}
		storeLeaderboardCache(ctx, svcCtx, key, loaded)
		return loaded, nil
	})
	if err != nil {
		return nil, err
	}
	return value.(*leaderboardSharedPayload), nil
}

func leaderboardCacheVersion(ctx context.Context, svcCtx *svc.ServiceContext, gameType int) (int64, bool) {
	if svcCtx == nil || svcCtx.Redis == nil {
		return 0, false
	}
	version, err := svcCtx.Redis.Get(ctx, leaderboardVersionKey(gameType)).Int64()
	if err == nil {
		return version, true
	}
	if err == redis.Nil {
		return 0, true
	}
	leaderboardCacheMetrics.RedisError(ctx)
	logx.WithContext(ctx).Errorf("读取排行榜缓存版本失败: gameType=%d err=%v", gameType, err)
	return 0, false
}

func leaderboardVersionKey(gameType int) string {
	if gameType <= 0 {
		gameType = 3
	}
	return fmt.Sprintf("leaderboard:version:%d", gameType)
}

func readLeaderboardCache(ctx context.Context, svcCtx *svc.ServiceContext, key string, recordMetrics bool) (*leaderboardSharedPayload, bool) {
	payload, err := svcCtx.Redis.Get(ctx, key).Bytes()
	if err == nil {
		var cached leaderboardSharedPayload
		if err := json.Unmarshal(payload, &cached); err == nil {
			if recordMetrics {
				leaderboardCacheMetrics.Hit(ctx)
			}
			return &cached, true
		}
		leaderboardCacheMetrics.DecodeError(ctx)
		if delErr := svcCtx.Redis.Del(ctx, key).Err(); delErr != nil {
			leaderboardCacheMetrics.WriteError(ctx)
		}
		return nil, false
	}
	if err == redis.Nil {
		if recordMetrics {
			leaderboardCacheMetrics.Miss(ctx)
		}
		return nil, false
	}
	leaderboardCacheMetrics.RedisError(ctx)
	logx.WithContext(ctx).Errorf("读取排行榜缓存失败: key=%s err=%v", key, err)
	return nil, false
}

func storeLeaderboardCache(ctx context.Context, svcCtx *svc.ServiceContext, key string, value *leaderboardSharedPayload) {
	if svcCtx == nil || svcCtx.Redis == nil || value == nil {
		return
	}
	payload, err := json.Marshal(value)
	if err != nil {
		leaderboardCacheMetrics.WriteError(ctx)
		return
	}
	if err := svcCtx.Redis.Set(ctx, key, payload, leaderboardCacheTTL).Err(); err != nil {
		leaderboardCacheMetrics.WriteError(ctx)
		logx.WithContext(ctx).Errorf("写入排行榜缓存失败: key=%s err=%v", key, err)
	}
}
