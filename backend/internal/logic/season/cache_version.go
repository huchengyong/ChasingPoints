package season

import (
	"context"
	"fmt"
	"time"

	"chasing_points/internal/observability"
	"chasing_points/internal/readcache"
	seasonx "chasing_points/internal/season"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

const (
	seasonLeaderboardCacheTTL = 30 * time.Second
	seasonInfoCacheTTL        = 5 * time.Minute
)

var (
	seasonCacheMetrics observability.CacheMetrics
	seasonCacheGroup   singleflight.Group
)

func BumpSeasonLeaderboardCacheVersion(ctx context.Context, svcCtx *svc.ServiceContext, seasonID int64, gameType int) error {
	return bumpSeasonCacheVersion(ctx, svcCtx, seasonLeaderboardVersionKey(seasonID, gameType), seasonID > 0)
}

func BumpSeasonLeaderboardProfileVersion(ctx context.Context, svcCtx *svc.ServiceContext) error {
	return bumpSeasonCacheVersion(ctx, svcCtx, "season:leaderboard:profile-version", true)
}

func BumpSeasonInfoCacheVersion(ctx context.Context, svcCtx *svc.ServiceContext) error {
	return bumpSeasonCacheVersion(ctx, svcCtx, "season:info:version", true)
}

func SeasonLeaderboardCacheVersion(ctx context.Context, svcCtx *svc.ServiceContext, seasonID int64, gameType int) (int64, error) {
	if seasonID <= 0 {
		return 0, nil
	}
	return seasonCacheVersion(ctx, svcCtx, seasonLeaderboardVersionKey(seasonID, gameType))
}

func loadSeasonLeaderboardResponse(ctx context.Context, svcCtx *svc.ServiceContext, seasonID int64, gameType, page, pageSize int, load func() (types.GetSeasonLeaderboardResp, error)) (types.GetSeasonLeaderboardResp, error) {
	version, _ := SeasonLeaderboardCacheVersion(ctx, svcCtx, seasonID, gameType)
	profileVersion, _ := seasonCacheVersion(ctx, svcCtx, "season:leaderboard:profile-version")
	key := fmt.Sprintf("season:leaderboard:v%d:p%d:season:%d:game:%d:page:%d:size:%d", version, profileVersion, seasonID, gameType, page, pageSize)
	var client *redis.Client
	if svcCtx != nil {
		client = svcCtx.Redis
	}
	return readcache.LoadJSON(ctx, client, key, seasonLeaderboardCacheTTL, &seasonCacheMetrics, &seasonCacheGroup, load)
}

func loadCurrentSeasonResponse(ctx context.Context, svcCtx *svc.ServiceContext, now time.Time, load func() (types.GetCurrentSeasonResp, error)) (types.GetCurrentSeasonResp, error) {
	version, _ := seasonCacheVersion(ctx, svcCtx, "season:info:version")
	window := "legacy"
	if svcCtx != nil {
		if policy, err := seasonx.NewPolicy(svcCtx.Config.SeasonLifecycle); err == nil && policy.Enabled {
			if current, started := policy.WindowAt(now); started {
				window = current.StartDate.Format(time.DateOnly)
			} else {
				window = "not-started"
			}
		}
	}
	key := fmt.Sprintf("season:current:v%d:window:%s", version, window)
	var client *redis.Client
	if svcCtx != nil {
		client = svcCtx.Redis
	}
	return readcache.LoadJSON(ctx, client, key, seasonInfoCacheTTL, &seasonCacheMetrics, &seasonCacheGroup, load)
}

func SeasonCacheMetrics() observability.CacheSnapshot {
	return seasonCacheMetrics.Snapshot()
}

func ResetSeasonCacheForTest() {
	seasonCacheMetrics.Reset()
	seasonCacheGroup = singleflight.Group{}
}

func seasonCacheVersion(ctx context.Context, svcCtx *svc.ServiceContext, key string) (int64, error) {
	if svcCtx == nil || svcCtx.Redis == nil {
		return 0, nil
	}
	version, err := svcCtx.Redis.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		seasonCacheMetrics.RedisError(ctx)
	}
	return version, err
}

func bumpSeasonCacheVersion(ctx context.Context, svcCtx *svc.ServiceContext, key string, enabled bool) error {
	if !enabled || svcCtx == nil || svcCtx.Redis == nil {
		return nil
	}
	if err := svcCtx.Redis.Incr(ctx, key).Err(); err != nil {
		seasonCacheMetrics.WriteError(ctx)
		return err
	}
	return nil
}

func seasonLeaderboardVersionKey(seasonID int64, gameType int) string {
	if gameType <= 0 {
		gameType = 3
	}
	return fmt.Sprintf("season:leaderboard:version:%d:%d", seasonID, gameType)
}
