package achievement

import (
	"errors"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/gorm"
)

type AchievementProgressEventInput struct {
	UserId      int64
	SourceType  string
	SourceId    int64
	GameType    int
	MetricKey   string
	MetricValue int
	OccurredAt  time.Time
}

type AchievementProgressService struct {
	svcCtx       *svc.ServiceContext
	titleService *TitleGrantService
}

type AchievementUnlockResult struct {
	Achievement     model.Achievement
	UserAchievement model.UserAchievement
	RewardTitle     *model.UserTitle
}

func NewAchievementProgressService(svcCtx *svc.ServiceContext) *AchievementProgressService {
	return &AchievementProgressService{
		svcCtx:       svcCtx,
		titleService: NewTitleGrantService(svcCtx),
	}
}

func (s *AchievementProgressService) AppendEvent(input AchievementProgressEventInput) (bool, error) {
	event := model.NewAchievementProgressEvent(
		input.UserId,
		input.SourceType,
		input.SourceId,
		input.GameType,
		input.MetricKey,
		input.MetricValue,
		input.OccurredAt,
	)

	return s.svcCtx.AchievementProgressEventModel.CreateIfAbsent(event)
}

func (s *AchievementProgressService) RefreshUserAchievements(userId int64) error {
	_, err := s.RefreshUserAchievementsWithSource(userId, "", 0)
	return err
}

func (s *AchievementProgressService) RefreshUserAchievementsWithSource(userId int64, sourceType string, sourceId int64) ([]AchievementUnlockResult, error) {
	var achievements []model.Achievement
	if err := s.svcCtx.DB.
		Where("status = ? AND metric_key <> ''", 1).
		Order("sort ASC, id ASC").
		Find(&achievements).Error; err != nil {
		return nil, err
	}

	results := make([]AchievementUnlockResult, 0)
	for _, achievement := range achievements {
		result, err := s.RefreshUserAchievementWithSource(userId, achievement, sourceType, sourceId)
		if err != nil {
			return nil, err
		}
		if result != nil {
			results = append(results, *result)
		}
	}
	return results, nil
}

func (s *AchievementProgressService) RefreshUserAchievement(userId int64, achievement model.Achievement) error {
	_, err := s.RefreshUserAchievementWithSource(userId, achievement, "", 0)
	return err
}

func (s *AchievementProgressService) RefreshUserAchievementWithSource(userId int64, achievement model.Achievement, sourceType string, sourceId int64) (*AchievementUnlockResult, error) {
	progress, err := s.aggregateProgress(userId, achievement)
	if err != nil {
		return nil, err
	}

	var ua model.UserAchievement
	err = s.svcCtx.DB.Where("user_id = ? AND achievement_id = ?", userId, achievement.Id).First(&ua).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	now := time.Now()
	shouldUnlock := progress >= achievement.Threshold
	newlyUnlocked := false
	if errors.Is(err, gorm.ErrRecordNotFound) {
		ua = model.UserAchievement{
			UserId:        userId,
			AchievementId: achievement.Id,
			Progress:      progress,
		}
		if shouldUnlock {
			ua.Unlocked = 1
			ua.UnlockedAt = &now
			ua.UnlockedSourceType = sourceType
			ua.UnlockedSourceId = sourceId
			newlyUnlocked = true
		}
		if err := s.svcCtx.DB.Create(&ua).Error; err != nil {
			return nil, err
		}
	} else {
		updates := map[string]any{"progress": progress}
		if ua.Unlocked == 0 && shouldUnlock {
			updates["unlocked"] = 1
			updates["unlocked_at"] = &now
			updates["unlocked_source_type"] = sourceType
			updates["unlocked_source_id"] = sourceId
		}
		query := s.svcCtx.DB.Model(&model.UserAchievement{}).Where("id = ?", ua.Id)
		if ua.Unlocked == 0 && shouldUnlock {
			query = query.Where("unlocked = 0")
		}
		result := query.Updates(updates)
		if result.Error != nil {
			return nil, result.Error
		}
		if ua.Unlocked == 0 && shouldUnlock {
			if result.RowsAffected > 0 {
				ua.Unlocked = 1
				ua.UnlockedAt = &now
				ua.UnlockedSourceType = sourceType
				ua.UnlockedSourceId = sourceId
				newlyUnlocked = true
			} else if err := s.svcCtx.DB.First(&ua, ua.Id).Error; err != nil {
				return nil, err
			}
		}
		ua.Progress = progress
	}

	if ua.Unlocked == 1 && ua.RewardGranted == 0 && achievement.RewardTitleKey != "" && achievement.RewardTitleName != "" {
		if err := s.titleService.GrantAchievementTitle(userId, achievement); err != nil {
			return nil, err
		}
		if err := s.svcCtx.DB.Model(&model.UserAchievement{}).
			Where("id = ? AND reward_granted = 0", ua.Id).
			Updates(map[string]any{
				"reward_granted":    1,
				"reward_granted_at": &now,
			}).Error; err != nil {
			return nil, err
		}
		ua.RewardGranted = 1
		ua.RewardGrantedAt = &now
	}

	if !newlyUnlocked {
		return nil, nil
	}

	var rewardTitle *model.UserTitle
	if achievement.RewardTitleKey != "" && achievement.RewardTitleName != "" {
		var title model.UserTitle
		if err := s.svcCtx.DB.Where(
			"user_id = ? AND source_type = ? AND source_ref_id = ?",
			userId,
			SourceTypeAchievement,
			achievement.Id,
		).First(&title).Error; err != nil {
			return nil, err
		}
		rewardTitle = &title
	}

	return &AchievementUnlockResult{
		Achievement:     achievement,
		UserAchievement: ua,
		RewardTitle:     rewardTitle,
	}, nil
}

func (s *AchievementProgressService) aggregateProgress(userId int64, achievement model.Achievement) (int, error) {
	query := s.svcCtx.DB.Model(&model.AchievementProgressEvent{}).
		Where("user_id = ? AND metric_key = ?", userId, achievement.MetricKey)
	if achievement.GameType > 0 {
		query = query.Where("game_type = ?", achievement.GameType)
	}

	var progress int
	switch achievement.ProgressMode {
	case ProgressModeMax:
		if err := query.Select("COALESCE(MAX(metric_value), 0)").Scan(&progress).Error; err != nil {
			return 0, err
		}
	default:
		if err := query.Select("COALESCE(SUM(metric_value), 0)").Scan(&progress).Error; err != nil {
			return 0, err
		}
	}
	return progress, nil
}
