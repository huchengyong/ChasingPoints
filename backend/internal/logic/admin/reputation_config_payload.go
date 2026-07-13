package admin

import (
	"errors"
	"fmt"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

var supportedReputationGameTypes = map[int]struct{}{
	1: {},
	2: {},
	3: {},
	4: {},
}

func buildAdminReputationConfigResp(cfg logicx.ReputationRuntimeConfig) *types.AdminReputationConfigResp {
	durationRules := make([]types.AdminReputationDurationRule, 0, len(cfg.DetectionRules.DurationRules))
	for _, rule := range cfg.DetectionRules.DurationRules {
		durationRules = append(durationRules, types.AdminReputationDurationRule{
			GameType:                rule.GameType,
			Enabled:                 rule.Enabled,
			MinMinutesPerRound:      rule.MinMinutesPerRound,
			MinTotalRounds:          rule.MinTotalRounds,
			MinTotalDurationMinutes: rule.MinTotalDurationMinutes,
			PenaltyScore:            rule.PenaltyScore,
		})
	}

	return &types.AdminReputationConfigResp{
		Code:    0,
		Success: true,
		Message: "success",
		BaseRules: types.AdminReputationBaseRules{
			MaxScore:         cfg.BaseRules.MaxScore,
			InitialScore:     cfg.BaseRules.InitialScore,
			BanThreshold:     cfg.BaseRules.BanThreshold,
			BanDurationHours: cfg.BaseRules.BanDurationHours,
			MinScore:         cfg.BaseRules.MinScore,
		},
		RecoveryRules: types.AdminReputationRecoveryRules{
			Enabled:         cfg.RecoveryRules.Enabled,
			RecoverPerHour:  cfg.RecoveryRules.RecoverPerHour,
			RecoverMaxScore: cfg.RecoveryRules.RecoverMaxScore,
		},
		DetectionRules: types.AdminReputationDetectionRules{
			DurationRules: durationRules,
			SameOpponentRule: types.AdminReputationSameOpponentRule{
				Enabled:             cfg.DetectionRules.SameOpponentRule.Enabled,
				WindowMinutes:       cfg.DetectionRules.SameOpponentRule.WindowMinutes,
				MaxMatches:          cfg.DetectionRules.SameOpponentRule.MaxMatches,
				PenaltyScore:        cfg.DetectionRules.SameOpponentRule.PenaltyScore,
				RequireSameGameType: cfg.DetectionRules.SameOpponentRule.RequireSameGameType,
			},
			StackPenaltiesPerMatch: cfg.DetectionRules.StackPenaltiesPerMatch,
		},
	}
}

func buildReputationConfigModel(req *types.AdminReputationConfigUpdateReq, updatedBy int64) *model.ReputationConfig {
	cfg := &model.ReputationConfig{
		ConfigKey: model.DefaultReputationConfigKey,
		UpdatedBy: updatedBy,
	}
	cfg.SetBaseRules(model.ReputationBaseRules{
		MaxScore:         req.BaseRules.MaxScore,
		InitialScore:     req.BaseRules.InitialScore,
		BanThreshold:     req.BaseRules.BanThreshold,
		BanDurationHours: req.BaseRules.BanDurationHours,
		MinScore:         req.BaseRules.MinScore,
	})
	cfg.SetRecoveryRules(model.ReputationRecoveryRules{
		Enabled:         req.RecoveryRules.Enabled,
		RecoverPerHour:  req.RecoveryRules.RecoverPerHour,
		RecoverMaxScore: req.RecoveryRules.RecoverMaxScore,
	})

	durationRules := make([]model.ReputationDurationRule, 0, len(req.DetectionRules.DurationRules))
	for _, rule := range req.DetectionRules.DurationRules {
		durationRules = append(durationRules, model.ReputationDurationRule{
			GameType:                rule.GameType,
			Enabled:                 rule.Enabled,
			MinMinutesPerRound:      rule.MinMinutesPerRound,
			MinTotalRounds:          rule.MinTotalRounds,
			MinTotalDurationMinutes: rule.MinTotalDurationMinutes,
			PenaltyScore:            rule.PenaltyScore,
		})
	}
	cfg.SetDetectionRules(model.ReputationDetectionRules{
		DurationRules: durationRules,
		SameOpponentRule: model.ReputationSameOpponentRule{
			Enabled:             req.DetectionRules.SameOpponentRule.Enabled,
			WindowMinutes:       req.DetectionRules.SameOpponentRule.WindowMinutes,
			MaxMatches:          req.DetectionRules.SameOpponentRule.MaxMatches,
			PenaltyScore:        req.DetectionRules.SameOpponentRule.PenaltyScore,
			RequireSameGameType: req.DetectionRules.SameOpponentRule.RequireSameGameType,
		},
		StackPenaltiesPerMatch: req.DetectionRules.StackPenaltiesPerMatch,
	})
	return cfg
}

func validateReputationConfigUpdateReq(req *types.AdminReputationConfigUpdateReq) error {
	if req == nil {
		return errors.New("请求不能为空")
	}

	base := req.BaseRules
	if base.MaxScore <= 0 {
		return errors.New("信誉满分必须大于 0")
	}
	if base.MinScore < 0 {
		return errors.New("信誉最低分不能小于 0")
	}
	if base.MinScore >= base.MaxScore {
		return errors.New("信誉最低分必须小于信誉满分")
	}
	if base.InitialScore < base.MinScore || base.InitialScore > base.MaxScore {
		return errors.New("新用户初始信誉必须落在最小分和满分之间")
	}
	if base.BanThreshold < base.MinScore || base.BanThreshold > base.MaxScore {
		return errors.New("禁赛阈值必须落在最小分和满分之间")
	}
	if base.InitialScore < base.BanThreshold {
		return errors.New("新用户初始信誉不能低于禁赛阈值")
	}
	if base.BanDurationHours < 0 {
		return errors.New("禁赛时长不能为负数")
	}

	recovery := req.RecoveryRules
	if recovery.RecoverPerHour < 0 {
		return errors.New("每小时恢复值不能为负数")
	}
	if recovery.Enabled && recovery.RecoverPerHour <= 0 {
		return errors.New("启用自然恢复时，每小时恢复值必须大于 0")
	}
	if recovery.RecoverMaxScore < base.MinScore || recovery.RecoverMaxScore > base.MaxScore {
		return errors.New("恢复上限必须落在最小分和满分之间")
	}

	detection := req.DetectionRules
	seenGameTypes := make(map[int]struct{}, len(detection.DurationRules))
	for _, rule := range detection.DurationRules {
		if _, ok := supportedReputationGameTypes[rule.GameType]; !ok {
			return fmt.Errorf("玩法 %d 不受支持", rule.GameType)
		}
		if _, exists := seenGameTypes[rule.GameType]; exists {
			return fmt.Errorf("玩法 %d 的时长规则重复", rule.GameType)
		}
		seenGameTypes[rule.GameType] = struct{}{}
		if rule.MinMinutesPerRound <= 0 {
			return fmt.Errorf("玩法 %d 的单局最短分钟数必须大于 0", rule.GameType)
		}
		if rule.MinTotalRounds <= 0 {
			return fmt.Errorf("玩法 %d 的最少总局数必须大于 0", rule.GameType)
		}
		if rule.MinTotalDurationMinutes <= 0 {
			return fmt.Errorf("玩法 %d 的最短总时长必须大于 0", rule.GameType)
		}
		if rule.PenaltyScore < 0 {
			return fmt.Errorf("玩法 %d 的扣分不能为负数", rule.GameType)
		}
		if rule.Enabled && rule.PenaltyScore <= 0 {
			return fmt.Errorf("玩法 %d 启用时，扣分必须大于 0", rule.GameType)
		}
	}

	sameOpponent := detection.SameOpponentRule
	if sameOpponent.WindowMinutes < 0 {
		return errors.New("同对手高频窗口不能为负数")
	}
	if sameOpponent.MaxMatches < 0 {
		return errors.New("同对手窗口内最大场次不能为负数")
	}
	if sameOpponent.PenaltyScore < 0 {
		return errors.New("同对手高频扣分不能为负数")
	}
	if sameOpponent.Enabled {
		if sameOpponent.WindowMinutes <= 0 {
			return errors.New("启用同对手高频规则时，窗口分钟数必须大于 0")
		}
		if sameOpponent.MaxMatches <= 0 {
			return errors.New("启用同对手高频规则时，最大场次必须大于 0")
		}
		if sameOpponent.PenaltyScore <= 0 {
			return errors.New("启用同对手高频规则时，扣分必须大于 0")
		}
	}

	return nil
}
