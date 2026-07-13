package achievement

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserAchievementsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取已解锁成就
func NewGetUserAchievementsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserAchievementsLogic {
	return &GetUserAchievementsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserAchievementsLogic) GetUserAchievements() (resp *types.GetUserAchievementsResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetUserAchievementsResp{Success: false}, nil
	}

	unlockedList, err := l.svcCtx.UserAchievementModel.FindUnlockedByUserId(userIdInt)
	if err != nil {
		l.Logger.Errorf("查询已解锁成就失败: %v", err)
		return &types.GetUserAchievementsResp{Success: false}, nil
	}

	achievements, err := l.svcCtx.AchievementModel.FindAll()
	if err != nil {
		l.Logger.Errorf("查询成就定义失败: %v", err)
		return &types.GetUserAchievementsResp{Success: false}, nil
	}

	achievementMap := make(map[int64]model.Achievement, len(achievements))
	for _, achievement := range achievements {
		achievementMap[achievement.Id] = achievement
	}

	list := make([]types.AchievementDef, 0, len(unlockedList))
	for _, unlocked := range unlockedList {
		achievement, exists := achievementMap[unlocked.AchievementId]
		if !exists {
			continue
		}

		item := types.AchievementDef{
			Id:          achievement.Id,
			Key:         achievement.Key,
			Name:        achievement.Name,
			Description: achievement.Description,
			Icon:        achievement.Icon,
			Category:    achievement.Category,
			Threshold:   achievement.Threshold,
			Progress:    unlocked.Progress,
			Unlocked:    true,
		}
		if unlocked.UnlockedAt != nil {
			item.UnlockedAt = unlocked.UnlockedAt.Format("2006-01-02 15:04:05")
		}
		list = append(list, item)
	}

	return &types.GetUserAchievementsResp{
		Success: true,
		Total:   len(list),
		List:    list,
	}, nil
}
