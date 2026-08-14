package user

import (
	"context"

	memberlogic "chasing_points/internal/logic/member"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserOverviewLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取个人主页概览
func NewGetUserOverviewLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserOverviewLogic {
	return &GetUserOverviewLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetUserOverviewLogic) GetUserOverview() (resp *types.GetUserOverviewResp, err error) {
	result := &types.GetUserOverviewResp{Availability: map[string]bool{}, PartialErrors: []types.ReadPartialError{}}
	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil || l.svcCtx == nil || l.svcCtx.UserModel == nil {
		return result, nil
	}
	user, err := currentUserFromRequest(l.ctx, l.svcCtx, userID)
	if err != nil || user == nil {
		return result, nil
	}
	result.Success = true

	if l.svcCtx.CompetitiveReadModelsEnabled() && l.svcCtx.CompetitiveReadModel != nil {
		stats, statsErr := l.svcCtx.CompetitiveReadModel.FindStats(userID, 0)
		if statsErr != nil {
			appendOverviewPartial(result, "stats", "统计读取失败")
			appendOverviewPartial(result, "competitive_revision", "竞技版本读取失败")
		} else if stats != nil {
			result.Stats = buildUserStatsFromSnapshot(stats)
			result.CompetitiveRevision = stats.Revision
			result.Availability["stats"] = true
			result.Availability["competitive_revision"] = true
		} else if l.svcCtx.MatchModel == nil {
			appendOverviewPartial(result, "stats", "统计读取失败")
			appendOverviewPartial(result, "competitive_revision", "竞技快照缺失")
		} else {
			legacyStats, legacyErr := NewGetUserStatsLogic(l.ctx, l.svcCtx).getLegacyUserStats(userID)
			if legacyErr != nil || legacyStats == nil || !legacyStats.Success {
				appendOverviewPartial(result, "stats", "统计读取失败")
			} else {
				result.Stats = legacyStats
				result.Availability["stats"] = true
			}
			appendOverviewPartial(result, "competitive_revision", "竞技快照缺失")
		}
	} else {
		if stats, statsErr := NewGetUserStatsLogic(l.ctx, l.svcCtx).GetUserStats(); statsErr != nil || stats == nil || !stats.Success {
			appendOverviewPartial(result, "stats", "统计读取失败")
		} else {
			result.Stats = stats
			result.Availability["stats"] = true
		}
		appendOverviewPartial(result, "competitive_revision", "竞技读模型尚未切换")
	}

	if reputation, reputationErr := NewGetUserReputationLogic(l.ctx, l.svcCtx).GetUserReputation(); reputationErr != nil || reputation == nil || !reputation.Success {
		appendOverviewPartial(result, "reputation", "信誉读取失败")
	} else {
		result.Reputation = reputation
		result.Availability["reputation"] = true
	}
	if member, memberErr := memberlogic.NewGetMemberStatusLogic(l.ctx, l.svcCtx).GetMemberStatus(); memberErr != nil || member == nil || !member.Success {
		appendOverviewPartial(result, "member", "会员读取失败")
	} else {
		result.MemberStatus = member
		result.Availability["member"] = true
	}
	if l.svcCtx.FavoriteVenueRewardConfigModel == nil || l.svcCtx.FavoriteVenueRewardRecordModel == nil || l.svcCtx.VenueModel == nil {
		appendOverviewPartial(result, "favorite_venue_reward", "常玩球馆奖励服务不可用")
	} else if reward, rewardErr := NewGetFavoriteVenueRewardStatusLogic(l.ctx, l.svcCtx).GetFavoriteVenueRewardStatus(); rewardErr != nil || reward == nil || !reward.Success {
		appendOverviewPartial(result, "favorite_venue_reward", "常玩球馆奖励读取失败")
	} else {
		result.FavoriteVenueReward = reward
		result.Availability["favorite_venue_reward"] = true
	}
	return result, nil
}

func appendOverviewPartial(resp *types.GetUserOverviewResp, scope, message string) {
	resp.Availability[scope] = false
	resp.PartialErrors = append(resp.PartialErrors, types.ReadPartialError{Scope: scope, Message: message})
}
