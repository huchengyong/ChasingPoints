package logic

import (
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
)

type ReputationRuntimeConfig struct {
	ConfigKey      string
	BaseRules      model.ReputationBaseRules
	RecoveryRules  model.ReputationRecoveryRules
	DetectionRules model.ReputationDetectionRules
}

type ReputationConfigService struct {
	svcCtx *svc.ServiceContext
}

func NewReputationConfigService(svcCtx *svc.ServiceContext) *ReputationConfigService {
	return &ReputationConfigService{svcCtx: svcCtx}
}

func (s *ReputationConfigService) GetConfig() (ReputationRuntimeConfig, error) {
	defaultConfig := model.DefaultReputationConfig()
	baseRules, _ := defaultConfig.BaseRules()
	recoveryRules, _ := defaultConfig.RecoveryRules()
	detectionRules, _ := defaultConfig.DetectionRules()
	fallback := ReputationRuntimeConfig{
		ConfigKey:      model.DefaultReputationConfigKey,
		BaseRules:      baseRules,
		RecoveryRules:  recoveryRules,
		DetectionRules: detectionRules,
	}

	if s == nil || s.svcCtx == nil || s.svcCtx.ReputationConfigModel == nil {
		return fallback, nil
	}

	cfg, err := s.svcCtx.ReputationConfigModel.FindByKey(model.DefaultReputationConfigKey)
	if err != nil {
		return fallback, err
	}
	if cfg == nil {
		return fallback, nil
	}

	baseRules, err = cfg.BaseRules()
	if err != nil {
		return fallback, err
	}
	recoveryRules, err = cfg.RecoveryRules()
	if err != nil {
		return fallback, err
	}
	detectionRules, err = cfg.DetectionRules()
	if err != nil {
		return fallback, err
	}

	return ReputationRuntimeConfig{
		ConfigKey:      cfg.ConfigKey,
		BaseRules:      baseRules,
		RecoveryRules:  recoveryRules,
		DetectionRules: detectionRules,
	}, nil
}
