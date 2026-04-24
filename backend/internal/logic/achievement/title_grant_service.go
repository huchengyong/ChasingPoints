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
	if achievement.RewardTitleKey == "" || achievement.RewardTitleName == "" {
		return nil
	}

	now := time.Now()
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
		GrantedAt:              &now,
	}

	return s.svcCtx.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "title_key"},
			{Name: "source_type"},
			{Name: "source_ref_id"},
		},
		DoNothing: true,
	}).Create(title).Error
}
