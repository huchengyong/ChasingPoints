package logic

import (
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
)

type MemberRightsRuntimeConfig struct {
	ConfigKey     string
	GrowthRules   model.MemberGrowthRulesConfig
	RankingRights model.MemberRankingRightsRulesConfig
}

type MemberRightsConfigService struct {
	svcCtx *svc.ServiceContext
}

func NewMemberRightsConfigService(svcCtx *svc.ServiceContext) *MemberRightsConfigService {
	return &MemberRightsConfigService{svcCtx: svcCtx}
}

func (s *MemberRightsConfigService) GetConfig() (MemberRightsRuntimeConfig, error) {
	defaultConfig := model.DefaultMemberRightsConfig()
	defaultGrowth, _ := defaultConfig.GrowthRules()
	defaultRights, _ := defaultConfig.RankingRightsRules()
	fallback := MemberRightsRuntimeConfig{
		ConfigKey:     model.DefaultMemberRightsConfigKey,
		GrowthRules:   defaultGrowth,
		RankingRights: defaultRights,
	}

	if s == nil || s.svcCtx == nil || s.svcCtx.MemberRightsConfigModel == nil {
		return fallback, nil
	}

	cfg, err := s.svcCtx.MemberRightsConfigModel.FindByKey(model.DefaultMemberRightsConfigKey)
	if err != nil {
		return fallback, err
	}
	if cfg == nil {
		return fallback, nil
	}

	growth, err := cfg.GrowthRules()
	if err != nil {
		return fallback, err
	}
	rights, err := cfg.RankingRightsRules()
	if err != nil {
		return fallback, err
	}

	return MemberRightsRuntimeConfig{
		ConfigKey:     cfg.ConfigKey,
		GrowthRules:   growth,
		RankingRights: rights,
	}, nil
}
