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
}

type AchievementProgressService struct {
	svcCtx       *svc.ServiceContext
	titleService *TitleGrantService
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
	)

	return s.svcCtx.AchievementProgressEventModel.CreateIfAbsent(event)
}

func (s *AchievementProgressService) RefreshUserAchievements(userId int64) error {
	var achievements []model.Achievement
	if err := s.svcCtx.DB.
		Where("status = ? AND metric_key <> ''", 1).
		Order("sort ASC, id ASC").
		Find(&achievements).Error; err != nil {
		return err
	}

	for _, achievement := range achievements {
		if err := s.RefreshUserAchievement(userId, achievement); err != nil {
			return err
		}
	}
	return nil
}

func (s *AchievementProgressService) RefreshUserAchievement(userId int64, achievement model.Achievement) error {
	progress, err := s.aggregateProgress(userId, achievement)
	if err != nil {
		return err
	}

	var ua model.UserAchievement
	err = s.svcCtx.DB.Where("user_id = ? AND achievement_id = ?", userId, achievement.Id).First(&ua).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	now := time.Now()
	shouldUnlock := progress >= achievement.Threshold
	if errors.Is(err, gorm.ErrRecordNotFound) {
		ua = model.UserAchievement{
			UserId:        userId,
			AchievementId: achievement.Id,
			Progress:      progress,
		}
		if shouldUnlock {
			ua.Unlocked = 1
			ua.UnlockedAt = &now
		}
		if err := s.svcCtx.DB.Create(&ua).Error; err != nil {
			return err
		}
	} else {
		updates := map[string]any{"progress": progress}
		if ua.Unlocked == 0 && shouldUnlock {
			updates["unlocked"] = 1
			updates["unlocked_at"] = &now
			ua.Unlocked = 1
			ua.UnlockedAt = &now
		}
		if err := s.svcCtx.DB.Model(&model.UserAchievement{}).
			Where("id = ?", ua.Id).
			Updates(updates).Error; err != nil {
			return err
		}
		ua.Progress = progress
	}

	if ua.Unlocked != 1 || ua.RewardGranted == 1 {
		return nil
	}
	if achievement.RewardTitleKey == "" || achievement.RewardTitleName == "" {
		return nil
	}
	if err := s.titleService.GrantAchievementTitle(userId, achievement); err != nil {
		return err
	}

	return s.svcCtx.DB.Model(&model.UserAchievement{}).
		Where("id = ? AND reward_granted = 0", ua.Id).
		Updates(map[string]any{
			"reward_granted":    1,
			"reward_granted_at": &now,
		}).Error
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
