package venue

import (
	"context"
	"testing"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestVenueNearbyCacheUsesLocationBucketsRadiusLimitAndVersion(t *testing.T) {
	miniRedis := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	svcCtx := &svc.ServiceContext{Redis: client}
	ResetVenueCacheForTest()
	loads := 0
	load := func() (types.GetVenueListResp, error) {
		loads++
		return types.GetVenueListResp{Success: true, Total: int64(loads)}, nil
	}
	params := func(latitude, longitude float64, radius, limit int) interface{} {
		return struct {
			Latitude, Longitude float64
			Radius, Limit       int
		}{venueLocationBucket(latitude), venueLocationBucket(longitude), radius, limit}
	}
	first, err := loadVenueCache(context.Background(), svcCtx, "nearby", params(22.54311, 114.05781, 5000, 3), venueNearbyCacheTTL, load)
	second, secondErr := loadVenueCache(context.Background(), svcCtx, "nearby", params(22.54314, 114.05784, 5000, 3), venueNearbyCacheTTL, load)
	third, thirdErr := loadVenueCache(context.Background(), svcCtx, "nearby", params(22.54314, 114.05784, 3000, 3), venueNearbyCacheTTL, load)
	if err != nil || secondErr != nil || thirdErr != nil || first.Total != 1 || second.Total != 1 || third.Total != 2 || loads != 2 {
		t.Fatalf("unexpected venue bucket cache: first=%+v second=%+v third=%+v loads=%d errors=%v/%v/%v", first, second, third, loads, err, secondErr, thirdErr)
	}
	if err := BumpVenueCacheVersion(context.Background(), svcCtx); err != nil {
		t.Fatalf("bump venue version: %v", err)
	}
	fourth, err := loadVenueCache(context.Background(), svcCtx, "nearby", params(22.54311, 114.05781, 5000, 3), venueNearbyCacheTTL, load)
	if err != nil || fourth.Total != 3 || loads != 3 {
		t.Fatalf("venue version must invalidate cached candidates: value=%+v loads=%d err=%v", fourth, loads, err)
	}
}
