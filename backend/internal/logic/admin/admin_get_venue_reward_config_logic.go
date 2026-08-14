package admin

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetVenueRewardConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取常玩球馆奖励配置
func NewAdminGetVenueRewardConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetVenueRewardConfigLogic {
	return &AdminGetVenueRewardConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *AdminGetVenueRewardConfigLogic) AdminGetVenueRewardConfig() (resp *types.AdminVenueRewardConfigResp, err error) {
	config, err := l.svcCtx.FavoriteVenueRewardConfigModel.FindByActivityKey(model.FavoriteVenueRewardActivityKey)
	if err != nil {
		l.Logger.Errorf("获取常玩球馆奖励配置失败: %v", err)
		return &types.AdminVenueRewardConfigResp{
			Code:    500,
			Success: false,
			Message: "获取奖励配置失败",
		}, nil
	}
	if config == nil {
		config = model.DefaultFavoriteVenueRewardConfig()
	}
	welcomeConfig, err := l.svcCtx.FavoriteVenueRewardConfigModel.FindByActivityKey(model.WelcomeMemberRewardActivityKey)
	if err != nil {
		l.Logger.Errorf("获取新用户会员奖励配置失败: %v", err)
		return &types.AdminVenueRewardConfigResp{
			Code:    500,
			Success: false,
			Message: "获取奖励配置失败",
		}, nil
	}
	if welcomeConfig == nil {
		welcomeConfig = model.DefaultWelcomeMemberRewardConfig()
	}

	resp = &types.AdminVenueRewardConfigResp{
		Code:                 0,
		Success:              true,
		Message:              "success",
		Enabled:              config.Enabled,
		PopupEnabled:         config.PopupEnabled,
		RewardDays:           config.RewardDays,
		NewUserWindowDays:    config.NewUserWindowDays,
		WelcomeRewardEnabled: welcomeConfig.Enabled,
		WelcomeRewardDays:    welcomeConfig.RewardDays,
	}
	if config.StartAt != nil {
		resp.StartAt = config.StartAt.Format("2006-01-02 15:04:05")
	}
	if config.EndAt != nil {
		resp.EndAt = config.EndAt.Format("2006-01-02 15:04:05")
	}
	return resp, nil
}
