package achievement

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAchievementDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 按 ID 获取成就详情
func NewGetAchievementDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAchievementDetailLogic {
	return &GetAchievementDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetAchievementDetailLogic) GetAchievementDetail(req *types.GetAchievementDetailReq) (resp *types.GetAchievementDetailResp, err error) {
	if req == nil || req.AchievementId <= 0 || l.svcCtx == nil || l.svcCtx.AchievementModel == nil || l.svcCtx.UserAchievementModel == nil {
		return &types.GetAchievementDetailResp{Success: false}, nil
	}
	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return &types.GetAchievementDetailResp{Success: false}, nil
	}
	achievement, err := l.svcCtx.AchievementModel.FindById(req.AchievementId)
	if err != nil || achievement == nil {
		return &types.GetAchievementDetailResp{Success: false}, nil
	}
	progress, err := l.svcCtx.UserAchievementModel.FindByUserAndAchievement(userID, achievement.Id)
	if err != nil {
		return &types.GetAchievementDetailResp{Success: false}, nil
	}
	return &types.GetAchievementDetailResp{Success: true, Achievement: achievementDefinition(achievement, progress)}, nil
}

func achievementDefinition(achievement *model.Achievement, progress *model.UserAchievement) *types.AchievementDef {
	if achievement == nil {
		return nil
	}
	result := &types.AchievementDef{
		Id:              achievement.Id,
		Key:             achievement.Key,
		Name:            achievement.Name,
		Description:     achievement.Description,
		Icon:            achievement.Icon,
		Category:        achievement.Category,
		GameType:        achievement.GameType,
		RewardTitleName: achievement.RewardTitleName,
		Threshold:       achievement.Threshold,
	}
	if progress != nil {
		result.Progress = progress.Progress
		result.Unlocked = progress.Unlocked == 1
		if progress.UnlockedAt != nil {
			result.UnlockedAt = progress.UnlockedAt.Format("2006-01-02 15:04:05")
		}
	}
	return result
}
