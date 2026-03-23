package model

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Achievement struct {
	Id          int64     `gorm:"primarykey"`
	Key         string    `gorm:"uniqueIndex;size:50;not null"`
	Name        string    `gorm:"size:100;not null"`
	Description string    `gorm:"size:500"`
	Icon        string    `gorm:"size:255"`
	Category    string    `gorm:"size:20;not null"`
	Threshold   int       `gorm:"not null;default:1"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

func (Achievement) TableName() string {
	return "achievements"
}

type UserAchievement struct {
	Id            int64 `gorm:"primarykey"`
	UserId        int64 `gorm:"not null;index"`
	AchievementId int64 `gorm:"not null;index"`
	Progress      int   `gorm:"not null;default:0"`
	Unlocked      int   `gorm:"not null;default:0"`
	UnlockedAt    *time.Time
	CreatedAt     time.Time `gorm:"autoCreateTime"`
}

func (UserAchievement) TableName() string {
	return "user_achievements"
}

type UserTitle struct {
	Id        int64     `gorm:"primarykey"`
	UserId    int64     `gorm:"not null;index"`
	TitleName string    `gorm:"size:50;not null"`
	Source    string    `gorm:"size:20;not null"`
	Equipped  int       `gorm:"not null;default:0"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (UserTitle) TableName() string {
	return "user_titles"
}

type AchievementModel struct {
	db *gorm.DB
}

func NewAchievementModel(db *gorm.DB) *AchievementModel {
	return &AchievementModel{db: db}
}

func (m *AchievementModel) FindAll() ([]Achievement, error) {
	var list []Achievement
	err := m.db.Order("id ASC").Find(&list).Error
	return list, err
}

func (m *AchievementModel) FindByCategory(category string) ([]Achievement, error) {
	var list []Achievement
	err := m.db.Where("category = ?", category).Order("id ASC").Find(&list).Error
	return list, err
}

func (m *AchievementModel) FindByIds(ids []int64) ([]Achievement, error) {
	var list []Achievement
	if len(ids) == 0 {
		return list, nil
	}

	err := m.db.Where("id IN ?", ids).Order("id ASC").Find(&list).Error
	return list, err
}

type UserAchievementModel struct {
	db *gorm.DB
}

func NewUserAchievementModel(db *gorm.DB) *UserAchievementModel {
	return &UserAchievementModel{db: db}
}

func (m *UserAchievementModel) FindByUserId(userId int64) ([]UserAchievement, error) {
	var list []UserAchievement
	err := m.db.Where("user_id = ?", userId).Order("id ASC").Find(&list).Error
	return list, err
}

func (m *UserAchievementModel) FindUnlockedByUserId(userId int64) ([]UserAchievement, error) {
	var list []UserAchievement
	err := m.db.Where("user_id = ? AND unlocked = 1", userId).Order("unlocked_at DESC, id DESC").Find(&list).Error
	return list, err
}

func (m *UserAchievementModel) FindUnlockedByUserIdBetween(userId int64, startDate, endDate time.Time, limit int) ([]UserAchievement, error) {
	var list []UserAchievement
	query := m.db.Where(
		"user_id = ? AND unlocked = 1 AND unlocked_at IS NOT NULL AND unlocked_at >= ? AND unlocked_at <= ?",
		userId,
		startDate,
		endDate,
	).Order("unlocked_at DESC, id DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&list).Error
	return list, err
}

func (m *UserAchievementModel) Unlock(userId, achievementId int64) error {
	now := time.Now()
	ua := &UserAchievement{
		UserId:        userId,
		AchievementId: achievementId,
		Unlocked:      1,
		UnlockedAt:    &now,
	}

	return m.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "achievement_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"unlocked":    1,
			"unlocked_at": &now,
		}),
	}).Create(ua).Error
}

type UserTitleModel struct {
	db *gorm.DB
}

func NewUserTitleModel(db *gorm.DB) *UserTitleModel {
	return &UserTitleModel{db: db}
}

func (m *UserTitleModel) FindByUserId(userId int64) ([]UserTitle, error) {
	var list []UserTitle
	err := m.db.Where("user_id = ?", userId).Order("id DESC").Find(&list).Error
	return list, err
}

func (m *UserTitleModel) Create(title *UserTitle) error {
	return m.db.Create(title).Error
}

func (m *UserTitleModel) EquipTitle(userId int64, titleId int64, equip bool) error {
	return m.db.Transaction(func(tx *gorm.DB) error {
		if equip {
			if err := tx.Model(&UserTitle{}).Where("user_id = ?", userId).Update("equipped", 0).Error; err != nil {
				return err
			}
			result := tx.Model(&UserTitle{}).Where("user_id = ? AND id = ?", userId, titleId).Update("equipped", 1)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return gorm.ErrRecordNotFound
			}
			return nil
		}

		result := tx.Model(&UserTitle{}).Where("user_id = ? AND id = ?", userId, titleId).Update("equipped", 0)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}
