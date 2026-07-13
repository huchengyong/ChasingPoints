package auth

import (
	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"time"
)

func resolveWelcomeMemberExpiresAt(svcCtx *svc.ServiceContext) *time.Time {
	config := model.DefaultWelcomeMemberRewardConfig()
	if svcCtx != nil && svcCtx.FavoriteVenueRewardConfigModel != nil {
		storedConfig, err := svcCtx.FavoriteVenueRewardConfigModel.FindByActivityKey(model.WelcomeMemberRewardActivityKey)
		if err == nil && storedConfig != nil {
			config = storedConfig
		}
	}
	if config == nil || !config.Enabled {
		return nil
	}

	rewardDays := config.RewardDays
	if rewardDays <= 0 {
		rewardDays = model.DefaultWelcomeMemberRewardConfig().RewardDays
	}
	expiresAt := logicx.NowUTC8().AddDate(0, 0, rewardDays)
	return &expiresAt
}
