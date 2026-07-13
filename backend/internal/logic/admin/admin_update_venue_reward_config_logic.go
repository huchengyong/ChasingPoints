package admin

import (
	"context"
	"strings"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

const adminRewardTimeLayout = "2006-01-02 15:04:05"

var adminRewardTimeLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

type AdminUpdateVenueRewardConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新常玩球馆奖励配置
func NewAdminUpdateVenueRewardConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminUpdateVenueRewardConfigLogic {
	return &AdminUpdateVenueRewardConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminUpdateVenueRewardConfigLogic) AdminUpdateVenueRewardConfig(req *types.AdminVenueRewardConfigUpdateReq) (resp *types.AdminWriteResp, err error) {
	if req == nil {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "请求不能为空",
		}, nil
	}
	if req.RewardDays <= 0 {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "奖励天数必须大于 0",
		}, nil
	}
	if req.WelcomeRewardDays <= 0 {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "新用户奖励天数必须大于 0",
		}, nil
	}

	startAt, err := parseAdminRewardOptionalTime(req.StartAt)
	if err != nil {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "开始时间格式错误",
		}, nil
	}
	endAt, err := parseAdminRewardOptionalTime(req.EndAt)
	if err != nil {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "结束时间格式错误",
		}, nil
	}
	if startAt != nil && endAt != nil && endAt.Before(*startAt) {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: "结束时间不能早于开始时间",
		}, nil
	}

	venueConfig, err := l.svcCtx.FavoriteVenueRewardConfigModel.FindByActivityKey(model.FavoriteVenueRewardActivityKey)
	if err != nil {
		l.Logger.Errorf("读取常玩球馆奖励配置失败: %v", err)
		return &types.AdminWriteResp{
			Code:    500,
			Success: false,
			Message: "保存奖励配置失败",
		}, nil
	}
	if venueConfig == nil {
		venueConfig = model.DefaultFavoriteVenueRewardConfig()
	}

	if err := l.svcCtx.FavoriteVenueRewardConfigModel.Upsert(&model.FavoriteVenueRewardConfig{
		ActivityKey:       model.FavoriteVenueRewardActivityKey,
		Enabled:           req.Enabled,
		PopupEnabled:      req.PopupEnabled,
		RewardDays:        req.RewardDays,
		NewUserWindowDays: venueConfig.NewUserWindowDays,
		StartAt:           startAt,
		EndAt:             endAt,
	}); err != nil {
		l.Logger.Errorf("更新常玩球馆奖励配置失败: %v", err)
		return &types.AdminWriteResp{
			Code:    500,
			Success: false,
			Message: "保存奖励配置失败",
		}, nil
	}

	welcomeConfig, err := l.svcCtx.FavoriteVenueRewardConfigModel.FindByActivityKey(model.WelcomeMemberRewardActivityKey)
	if err != nil {
		l.Logger.Errorf("读取新用户会员奖励配置失败: %v", err)
		return &types.AdminWriteResp{
			Code:    500,
			Success: false,
			Message: "保存奖励配置失败",
		}, nil
	}
	if welcomeConfig == nil {
		welcomeConfig = model.DefaultWelcomeMemberRewardConfig()
	}

	if err := l.svcCtx.FavoriteVenueRewardConfigModel.Upsert(&model.FavoriteVenueRewardConfig{
		ActivityKey:       model.WelcomeMemberRewardActivityKey,
		Enabled:           req.WelcomeRewardEnabled,
		PopupEnabled:      welcomeConfig.PopupEnabled,
		RewardDays:        req.WelcomeRewardDays,
		NewUserWindowDays: welcomeConfig.NewUserWindowDays,
		StartAt:           welcomeConfig.StartAt,
		EndAt:             welcomeConfig.EndAt,
	}); err != nil {
		l.Logger.Errorf("更新新用户会员奖励配置失败: %v", err)
		return &types.AdminWriteResp{
			Code:    500,
			Success: false,
			Message: "保存奖励配置失败",
		}, nil
	}

	return &types.AdminWriteResp{
		Code:    0,
		Success: true,
		Message: "保存成功",
	}, nil
}

func parseAdminRewardOptionalTime(raw string) (*time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := time.ParseInLocation(adminRewardTimeLayout, trimmed, adminRewardTimeLocation)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
