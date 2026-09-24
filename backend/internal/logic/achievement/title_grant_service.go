package achievement

import (
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/gorm/clause"
)

type TitleGrantService struct {
	svcCtx *svc.ServiceContext
}

func NewTitleGrantService(svcCtx *svc.ServiceContext) *TitleGrantService {
	return &TitleGrantService{svcCtx: svcCtx}
}

func (s *TitleGrantService) GrantAchievementTitle(userId int64, achievement model.Achievement) error {
	_, err := s.GrantAchievementTitleAt(userId, achievement, time.Now())
	return err
}

func (s *TitleGrantService) GrantAchievementTitleAt(userId int64, achievement model.Achievement, grantedAt time.Time) (bool, error) {
	if achievement.RewardTitleKey == "" || achievement.RewardTitleName == "" {
		return false, nil
	}
	if grantedAt.IsZero() {
		grantedAt = time.Now()
	}

	achievementId := achievement.Id
	title := &model.UserTitle{
		UserId:                 userId,
		TitleKey:               achievement.RewardTitleKey,
		TitleName:              achievement.RewardTitleName,
		Source:                 SourceTypeAchievement,
		SourceType:             SourceTypeAchievement,
		SourceRefId:            achievement.Id,
		SourceRefName:          achievement.Name,
		GrantedByAchievementId: &achievementId,
		GrantedAt:              &grantedAt,
	}

	result := s.svcCtx.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "title_key"},
			{Name: "source_type"},
			{Name: "source_ref_id"},
		},
		DoNothing: true,
	}).Create(title)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}
