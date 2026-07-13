package admin

import (
	"errors"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func buildAdminMemberRightsConfigResp(cfg logicx.MemberRightsRuntimeConfig) *types.AdminMemberRightsConfigResp {
	return &types.AdminMemberRightsConfigResp{
		Code:    0,
		Success: true,
		Message: "success",
		GrowthRules: types.AdminMemberGrowthRules{
			PointsPerCompletedMatch: cfg.GrowthRules.PointsPerCompletedMatch,
			DailyCap:                cfg.GrowthRules.DailyCap,
			LevelThresholdLv2:       cfg.GrowthRules.LevelThresholdLv2,
			LevelThresholdLv3:       cfg.GrowthRules.LevelThresholdLv3,
			LevelThresholdLv4:       cfg.GrowthRules.LevelThresholdLv4,
			LevelThresholdLv5:       cfg.GrowthRules.LevelThresholdLv5,
			ExpireStrategy:          cfg.GrowthRules.ExpireStrategy,
		},
		RankingRights: types.AdminMemberRankingRightsRules{
			OrdinaryUserAchievementEnabled: cfg.RankingRights.OrdinaryUserAchievementEnabled,
			DailyCap:                       cfg.RankingRights.DailyCap,
			DailyPositiveCap:               cfg.RankingRights.DailyPositiveCap,
			Break50Score:                   cfg.RankingRights.Break50Score,
			GoldenBreakScore:               cfg.RankingRights.GoldenBreakScore,
			BreakAndRunScore:               cfg.RankingRights.BreakAndRunScore,
			RunOutScore:                    cfg.RankingRights.RunOutScore,
			Break100Score:                  cfg.RankingRights.Break100Score,
			NineOnBreakScore:               cfg.RankingRights.NineOnBreakScore,
			Break147Score:                  cfg.RankingRights.Break147Score,
			Level1Multiplier:               cfg.RankingRights.Level1Multiplier,
			Level2Multiplier:               cfg.RankingRights.Level2Multiplier,
			Level3Multiplier:               cfg.RankingRights.Level3Multiplier,
			Level4Multiplier:               cfg.RankingRights.Level4Multiplier,
			Level5Multiplier:               cfg.RankingRights.Level5Multiplier,
		},
	}
}

func buildMemberRightsConfigModel(req *types.AdminMemberRightsConfigUpdateReq, updatedBy int64) *model.MemberRightsConfig {
	cfg := &model.MemberRightsConfig{
		ConfigKey: model.DefaultMemberRightsConfigKey,
		UpdatedBy: updatedBy,
	}
	cfg.SetGrowthRules(model.MemberGrowthRulesConfig{
		PointsPerCompletedMatch: req.GrowthRules.PointsPerCompletedMatch,
		DailyCap:                req.GrowthRules.DailyCap,
		LevelThresholdLv2:       req.GrowthRules.LevelThresholdLv2,
		LevelThresholdLv3:       req.GrowthRules.LevelThresholdLv3,
		LevelThresholdLv4:       req.GrowthRules.LevelThresholdLv4,
		LevelThresholdLv5:       req.GrowthRules.LevelThresholdLv5,
		ExpireStrategy:          req.GrowthRules.ExpireStrategy,
	})
	cfg.SetRankingRightsRules(model.MemberRankingRightsRulesConfig{
		OrdinaryUserAchievementEnabled: req.RankingRights.OrdinaryUserAchievementEnabled,
		DailyCap:                       req.RankingRights.DailyCap,
		DailyPositiveCap:               req.RankingRights.DailyPositiveCap,
		Break50Score:                   req.RankingRights.Break50Score,
		GoldenBreakScore:               req.RankingRights.GoldenBreakScore,
		BreakAndRunScore:               req.RankingRights.BreakAndRunScore,
		RunOutScore:                    req.RankingRights.RunOutScore,
		Break100Score:                  req.RankingRights.Break100Score,
		NineOnBreakScore:               req.RankingRights.NineOnBreakScore,
		Break147Score:                  req.RankingRights.Break147Score,
		Level1Multiplier:               req.RankingRights.Level1Multiplier,
		Level2Multiplier:               req.RankingRights.Level2Multiplier,
		Level3Multiplier:               req.RankingRights.Level3Multiplier,
		Level4Multiplier:               req.RankingRights.Level4Multiplier,
		Level5Multiplier:               req.RankingRights.Level5Multiplier,
	})
	return cfg
}

func validateMemberRightsConfigUpdateReq(req *types.AdminMemberRightsConfigUpdateReq) error {
	if req == nil {
		return errors.New("请求不能为空")
	}
	if req.GrowthRules.PointsPerCompletedMatch <= 0 {
		return errors.New("每场成长值必须大于 0")
	}
	if req.GrowthRules.DailyCap <= 0 {
		return errors.New("每日成长上限必须大于 0")
	}
	if req.GrowthRules.LevelThresholdLv2 <= 0 ||
		req.GrowthRules.LevelThresholdLv2 >= req.GrowthRules.LevelThresholdLv3 ||
		req.GrowthRules.LevelThresholdLv3 >= req.GrowthRules.LevelThresholdLv4 ||
		req.GrowthRules.LevelThresholdLv4 >= req.GrowthRules.LevelThresholdLv5 {
		return errors.New("会员成长等级门槛必须严格递增")
	}
	if req.GrowthRules.ExpireStrategy != "" && req.GrowthRules.ExpireStrategy != "freeze_preserve" {
		return errors.New("会员成长到期策略不支持")
	}
	if req.RankingRights.DailyCap <= 0 {
		return errors.New("会员特殊战绩日上限必须大于 0")
	}
	if req.RankingRights.DailyPositiveCap <= 0 {
		return errors.New("每日总正向上限必须大于 0")
	}
	scores := []int{
		req.RankingRights.Break50Score,
		req.RankingRights.GoldenBreakScore,
		req.RankingRights.BreakAndRunScore,
		req.RankingRights.RunOutScore,
		req.RankingRights.Break100Score,
		req.RankingRights.NineOnBreakScore,
		req.RankingRights.Break147Score,
	}
	for _, score := range scores {
		if score < 0 {
			return errors.New("特殊战绩分值不能为负数")
		}
	}
	multipliers := []int{
		req.RankingRights.Level1Multiplier,
		req.RankingRights.Level2Multiplier,
		req.RankingRights.Level3Multiplier,
		req.RankingRights.Level4Multiplier,
		req.RankingRights.Level5Multiplier,
	}
	for i, multiplier := range multipliers {
		if multiplier <= 0 {
			return errors.New("会员等级倍率必须大于 0")
		}
		if i > 0 && multiplier < multipliers[i-1] {
			return errors.New("会员等级倍率必须递增或持平")
		}
	}
	return nil
}
