package user

import (
	"context"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserReputationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUserReputationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserReputationLogic {
	return &GetUserReputationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserReputationLogic) GetUserReputation() (*types.GetUserReputationResp, error) {
	if l == nil || l.svcCtx == nil || l.svcCtx.UserReputationProfileModel == nil {
		return &types.GetUserReputationResp{Success: false}, nil
	}

	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户信誉时解析用户ID失败: %v", err)
		return &types.GetUserReputationResp{Success: false}, nil
	}

	cfg, err := logicx.NewReputationConfigService(l.svcCtx).GetConfig()
	if err != nil {
		l.Logger.Errorf("获取用户信誉配置失败: userID=%d err=%v", userID, err)
		return &types.GetUserReputationResp{Success: false}, nil
	}

	profile, err := l.svcCtx.UserReputationProfileModel.FindByUserID(userID)
	if err != nil {
		l.Logger.Errorf("获取用户信誉失败: userID=%d err=%v", userID, err)
		return &types.GetUserReputationResp{Success: false}, nil
	}

	now := logicx.NowUTC8()
	if profile == nil {
		return buildUserReputationResp(normalizedUserDisplayInitialScore(cfg.BaseRules), nil, now), nil
	}

	displayProfile := cloneUserReputationProfile(profile)
	applyUserDisplayReputationRecovery(displayProfile, cfg.BaseRules, cfg.RecoveryRules, now)
	return buildUserReputationResp(displayProfile.ReputationScore, displayProfile.BanUntil, now), nil
}

func cloneUserReputationProfile(profile *model.UserReputationProfile) *model.UserReputationProfile {
	if profile == nil {
		return nil
	}

	cloned := *profile
	if profile.LastRecoveredAt != nil {
		lastRecoveredAt := *profile.LastRecoveredAt
		cloned.LastRecoveredAt = &lastRecoveredAt
	}
	if profile.LastPenalizedAt != nil {
		lastPenalizedAt := *profile.LastPenalizedAt
		cloned.LastPenalizedAt = &lastPenalizedAt
	}
	if profile.BanUntil != nil {
		banUntil := *profile.BanUntil
		cloned.BanUntil = &banUntil
	}
	return &cloned
}
