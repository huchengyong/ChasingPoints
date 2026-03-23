package logic

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAchievementListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取成就列表
func NewGetAchievementListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAchievementListLogic {
	return &GetAchievementListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAchievementListLogic) GetAchievementList(req *types.GetAchievementListReq) (resp *types.GetAchievementListResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetAchievementListResp{Success: false}, nil
	}

	var achievements []model.Achievement
	if req.Category != "" {
		achievements, err = l.svcCtx.AchievementModel.FindByCategory(req.Category)
	} else {
		achievements, err = l.svcCtx.AchievementModel.FindAll()
	}
	if err != nil {
		l.Logger.Errorf("查询成就定义失败: %v", err)
		return &types.GetAchievementListResp{Success: false}, nil
	}

	userAchievements, err := l.svcCtx.UserAchievementModel.FindByUserId(userIdInt)
	if err != nil {
		l.Logger.Errorf("查询用户成就进度失败: %v", err)
		return &types.GetAchievementListResp{Success: false}, nil
	}

	progressMap := make(map[int64]model.UserAchievement, len(userAchievements))
	for _, item := range userAchievements {
		progressMap[item.AchievementId] = item
	}

	list := make([]types.AchievementDef, 0, len(achievements))
	for _, achievement := range achievements {
		progress := 0
		unlocked := false
		unlockedAt := ""

		if ua, exists := progressMap[achievement.Id]; exists {
			progress = ua.Progress
			unlocked = ua.Unlocked == 1
			if ua.UnlockedAt != nil {
				unlockedAt = ua.UnlockedAt.Format("2006-01-02 15:04:05")
			}
		}

		list = append(list, types.AchievementDef{
			Id:          achievement.Id,
			Key:         achievement.Key,
			Name:        achievement.Name,
			Description: achievement.Description,
			Icon:        achievement.Icon,
			Category:    achievement.Category,
			Threshold:   achievement.Threshold,
			Progress:    progress,
			Unlocked:    unlocked,
			UnlockedAt:  unlockedAt,
		})
	}

	return &types.GetAchievementListResp{
		Success: true,
		List:    list,
	}, nil
}
