package logic

import (
	"context"
	"time"

	"chasing_points/internal/logic/staticread"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
)

const (
	ReputationRuntimeConfigCacheKey   = "runtime-config:reputation"
	MemberRightsRuntimeConfigCacheKey = "runtime-config:member-rights"
	FavoriteVenueRewardConfigCacheKey = "runtime-config:favorite-venue-reward"
	runtimeConfigCacheTTL             = 5 * time.Minute
)

func runtimeConfigContext(svcCtx *svc.ServiceContext) context.Context {
	if svcCtx != nil && svcCtx.DB != nil && svcCtx.DB.Statement != nil && svcCtx.DB.Statement.Context != nil {
		return svcCtx.DB.Statement.Context
	}
	return context.Background()
}

func loadRuntimeConfig[T any](svcCtx *svc.ServiceContext, key string, load func() (T, error)) (T, error) {
	return staticread.LoadTTL(runtimeConfigContext(svcCtx), svcCtx, key, runtimeConfigCacheTTL, load)
}

func InvalidateRuntimeConfigCache(ctx context.Context, svcCtx *svc.ServiceContext, key string) {
	staticread.Invalidate(ctx, svcCtx, key)
}

func LoadFavoriteVenueRewardConfig(svcCtx *svc.ServiceContext) (*model.FavoriteVenueRewardConfig, error) {
	fallback := model.DefaultFavoriteVenueRewardConfig()
	if svcCtx == nil || svcCtx.FavoriteVenueRewardConfigModel == nil {
		return fallback, nil
	}
	return loadRuntimeConfig(svcCtx, FavoriteVenueRewardConfigCacheKey, func() (*model.FavoriteVenueRewardConfig, error) {
		config, err := svcCtx.FavoriteVenueRewardConfigModel.FindByActivityKey(model.FavoriteVenueRewardActivityKey)
		if err != nil {
			return nil, err
		}
		if config == nil {
			return model.DefaultFavoriteVenueRewardConfig(), nil
		}
		return config, nil
	})
}
