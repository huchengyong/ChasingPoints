package model

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Achievement struct {
	Id              int64     `gorm:"primarykey"`
	Key             string    `gorm:"uniqueIndex:uk_key;size:64;not null"`
	Name            string    `gorm:"size:128;not null"`
	Description     string    `gorm:"size:512;not null;default:''"`
	Icon            string    `gorm:"size:512;not null;default:''"`
	Category        string    `gorm:"size:32;not null;default:''"`
	GameType        int       `gorm:"not null;default:0"`
	MetricKey       string    `gorm:"size:64;not null;default:''"`
	ProgressMode    string    `gorm:"size:32;not null;default:'sum'"`
	RewardTitleKey  string    `gorm:"size:64;not null;default:''"`
	RewardTitleName string    `gorm:"size:64;not null;default:''"`
	Sort            int       `gorm:"not null;default:0"`
	Status          int       `gorm:"not null;default:1"`
	Threshold       int       `gorm:"not null;default:1"`
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
}

func (Achievement) TableName() string {
	return "achievements"
}

type UserAchievement struct {
	Id              int64 `gorm:"primarykey"`
	UserId          int64 `gorm:"not null;index;uniqueIndex:uk_user_achievement,priority:1"`
	AchievementId   int64 `gorm:"not null;index;uniqueIndex:uk_user_achievement,priority:2"`
	Progress        int   `gorm:"not null;default:0"`
	Unlocked        int   `gorm:"not null;default:0"`
	RewardGranted   int   `gorm:"not null;default:0"`
	UnlockedAt      *time.Time
	RewardGrantedAt *time.Time
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
}

func (UserAchievement) TableName() string {
	return "user_achievements"
}

type UserTitle struct {
	Id                     int64  `gorm:"primarykey"`
	UserId                 int64  `gorm:"not null;index;uniqueIndex:uk_user_title_source,priority:1"`
	TitleKey               string `gorm:"size:64;not null;default:'';uniqueIndex:uk_user_title_source,priority:2"`
	TitleName              string `gorm:"size:64;not null"`
	Source                 string `gorm:"size:32;not null;default:''"`
	SourceType             string `gorm:"size:32;not null;default:'';uniqueIndex:uk_user_title_source,priority:3"`
	SourceRefId            int64  `gorm:"type:bigint unsigned;not null;default:0;uniqueIndex:uk_user_title_source,priority:4"`
	SourceRefName          string `gorm:"size:128;not null;default:''"`
	GrantedByAchievementId *int64 `gorm:"type:bigint unsigned"`
	Equipped               int    `gorm:"not null;default:0"`
	EquippedAt             *time.Time
	GrantedAt              *time.Time
	CreatedAt              time.Time `gorm:"autoCreateTime"`
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
			if err := tx.Model(&UserTitle{}).Where("user_id = ?", userId).Updates(map[string]any{
				"equipped":    0,
				"equipped_at": nil,
			}).Error; err != nil {
				return err
			}
			now := time.Now()
			result := tx.Model(&UserTitle{}).Where("user_id = ? AND id = ?", userId, titleId).Updates(map[string]any{
				"equipped":    1,
				"equipped_at": &now,
			})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return gorm.ErrRecordNotFound
			}
			return nil
		}

		result := tx.Model(&UserTitle{}).Where("user_id = ? AND id = ?", userId, titleId).Updates(map[string]any{
			"equipped":    0,
			"equipped_at": nil,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

type AchievementProgressEvent struct {
	Id          int64     `gorm:"primarykey"`
	UserId      int64     `gorm:"not null;index:idx_achievement_progress_events_user_id;uniqueIndex:uk_user_source_metric,priority:1"`
	SourceType  string    `gorm:"size:32;not null;default:'';uniqueIndex:uk_user_source_metric,priority:2"`
	SourceId    int64     `gorm:"type:bigint unsigned;not null;default:0;uniqueIndex:uk_user_source_metric,priority:3"`
	GameType    int       `gorm:"not null;default:0;index:idx_achievement_progress_events_game_type"`
	MetricKey   string    `gorm:"size:64;not null;default:'';uniqueIndex:uk_user_source_metric,priority:4"`
	MetricValue int       `gorm:"not null;default:0"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

func (AchievementProgressEvent) TableName() string {
	return "achievement_progress_events"
}

func NewAchievementProgressEvent(userId int64, sourceType string, sourceId int64, gameType int, metricKey string, metricValue int) *AchievementProgressEvent {
	return &AchievementProgressEvent{
		UserId:      userId,
		SourceType:  sourceType,
		SourceId:    sourceId,
		GameType:    gameType,
		MetricKey:   metricKey,
		MetricValue: metricValue,
	}
}

type AchievementProgressEventModel struct {
	db *gorm.DB
}

func NewAchievementProgressEventModel(db *gorm.DB) *AchievementProgressEventModel {
	return &AchievementProgressEventModel{db: db}
}

func (m *AchievementProgressEventModel) CreateIfAbsent(event *AchievementProgressEvent) (bool, error) {
	result := m.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "source_type"},
			{Name: "source_id"},
			{Name: "metric_key"},
		},
		DoNothing: true,
	}).Create(event)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (m *AchievementProgressEventModel) FindBySourceMetric(userId int64, sourceType string, sourceId int64, metricKey string) (*AchievementProgressEvent, error) {
	var event AchievementProgressEvent
	err := m.db.Where(
		"user_id = ? AND source_type = ? AND source_id = ? AND metric_key = ?",
		userId,
		sourceType,
		sourceId,
		metricKey,
	).First(&event).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &event, err
}
