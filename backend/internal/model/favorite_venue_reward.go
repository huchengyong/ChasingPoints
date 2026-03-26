package model

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

const FavoriteVenueRewardActivityKey = "favorite_venue_member_reward"

func DefaultFavoriteVenueRewardConfig() *FavoriteVenueRewardConfig {
	return &FavoriteVenueRewardConfig{
		ActivityKey:       FavoriteVenueRewardActivityKey,
		Enabled:           false,
		PopupEnabled:      false,
		RewardDays:        30,
		NewUserWindowDays: 7,
	}
}

func FavoriteVenueRewardIsWithinWindow(userCreatedAt, submittedAt time.Time, windowDays int) bool {
	if userCreatedAt.IsZero() || submittedAt.IsZero() {
		return false
	}
	if windowDays <= 0 {
		windowDays = 7
	}
	deadline := userCreatedAt.AddDate(0, 0, windowDays)
	return !submittedAt.After(deadline)
}

func FavoriteVenueRewardWindowExpired(userCreatedAt, now time.Time, windowDays int) bool {
	if userCreatedAt.IsZero() {
		return true
	}
	if windowDays <= 0 {
		windowDays = 7
	}
	return now.After(userCreatedAt.AddDate(0, 0, windowDays))
}

func FavoriteVenueRewardConfigIsActive(config *FavoriteVenueRewardConfig, now time.Time) bool {
	if config == nil || !config.Enabled {
		return false
	}
	if config.StartAt != nil && now.Before(*config.StartAt) {
		return false
	}
	if config.EndAt != nil && now.After(*config.EndAt) {
		return false
	}
	return true
}

type FavoriteVenueRewardConfig struct {
	Id                int64      `gorm:"primarykey" json:"id"`
	ActivityKey       string     `gorm:"size:64;not null;uniqueIndex" json:"activity_key"`
	Enabled           bool       `gorm:"not null;default:false" json:"enabled"`
	PopupEnabled      bool       `gorm:"not null;default:false" json:"popup_enabled"`
	RewardDays        int        `gorm:"not null;default:30" json:"reward_days"`
	NewUserWindowDays int        `gorm:"not null;default:7" json:"new_user_window_days"`
	StartAt           *time.Time `json:"start_at"`
	EndAt             *time.Time `json:"end_at"`
	CreatedAt         time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (FavoriteVenueRewardConfig) TableName() string {
	return "favorite_venue_reward_configs"
}

type FavoriteVenueRewardRecord struct {
	Id                    int64      `gorm:"primarykey" json:"id"`
	ActivityKey           string     `gorm:"size:64;not null;uniqueIndex:uniq_activity_user,priority:1;uniqueIndex:uniq_activity_venue,priority:1" json:"activity_key"`
	UserId                int64      `gorm:"not null;uniqueIndex:uniq_activity_user,priority:2;index" json:"user_id"`
	VenueId               int64      `gorm:"not null;uniqueIndex:uniq_activity_venue,priority:2;index" json:"venue_id"`
	RewardDays            int        `gorm:"not null;default:30" json:"reward_days"`
	MemberExpiresAtBefore *time.Time `json:"member_expires_at_before"`
	MemberExpiresAtAfter  time.Time  `gorm:"not null" json:"member_expires_at_after"`
	GrantedAt             time.Time  `gorm:"not null" json:"granted_at"`
	CreatedAt             time.Time  `gorm:"autoCreateTime" json:"created_at"`
}

func (FavoriteVenueRewardRecord) TableName() string {
	return "favorite_venue_reward_records"
}

type FavoriteVenueRewardRecordAdminItem struct {
	Id                    int64      `json:"id"`
	UserId                int64      `json:"user_id"`
	UserNickname          string     `json:"user_nickname"`
	UserPhone             string     `json:"user_phone"`
	VenueId               int64      `json:"venue_id"`
	VenueName             string     `json:"venue_name"`
	RewardDays            int        `json:"reward_days"`
	MemberExpiresAtBefore *time.Time `json:"member_expires_at_before"`
	MemberExpiresAtAfter  time.Time  `json:"member_expires_at_after"`
	GrantedAt             time.Time  `json:"granted_at"`
}

type FavoriteVenueRewardConfigModel struct {
	db *gorm.DB
}

func NewFavoriteVenueRewardConfigModel(db *gorm.DB) *FavoriteVenueRewardConfigModel {
	_ = db.AutoMigrate(&FavoriteVenueRewardConfig{})
	return &FavoriteVenueRewardConfigModel{db: db}
}

func (m *FavoriteVenueRewardConfigModel) FindByActivityKey(activityKey string) (*FavoriteVenueRewardConfig, error) {
	if m == nil || m.db == nil {
		return nil, errors.New("favorite venue reward config db is nil")
	}

	var config FavoriteVenueRewardConfig
	result := m.db.Where("activity_key = ?", strings.TrimSpace(activityKey)).Limit(1).Find(&config)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &config, nil
}

func (m *FavoriteVenueRewardConfigModel) Upsert(config *FavoriteVenueRewardConfig) error {
	if m == nil || m.db == nil {
		return errors.New("favorite venue reward config db is nil")
	}
	if config == nil {
		return errors.New("favorite venue reward config is nil")
	}

	activityKey := strings.TrimSpace(config.ActivityKey)
	if activityKey == "" {
		return errors.New("activity key is required")
	}

	existing, err := m.FindByActivityKey(activityKey)
	if err != nil {
		return err
	}

	if existing == nil {
		config.ActivityKey = activityKey
		return m.db.Create(config).Error
	}

	updates := map[string]interface{}{
		"enabled":              config.Enabled,
		"popup_enabled":        config.PopupEnabled,
		"reward_days":          config.RewardDays,
		"new_user_window_days": config.NewUserWindowDays,
		"start_at":             config.StartAt,
		"end_at":               config.EndAt,
	}
	return m.db.Model(&FavoriteVenueRewardConfig{}).
		Where("id = ?", existing.Id).
		Updates(updates).Error
}

type FavoriteVenueRewardRecordModel struct {
	db *gorm.DB
}

func NewFavoriteVenueRewardRecordModel(db *gorm.DB) *FavoriteVenueRewardRecordModel {
	_ = db.AutoMigrate(&FavoriteVenueRewardRecord{})
	return &FavoriteVenueRewardRecordModel{db: db}
}

func (m *FavoriteVenueRewardRecordModel) Create(record *FavoriteVenueRewardRecord) error {
	return m.CreateWithTx(nil, record)
}

func (m *FavoriteVenueRewardRecordModel) CreateWithTx(tx *gorm.DB, record *FavoriteVenueRewardRecord) error {
	if record == nil {
		return errors.New("favorite venue reward record is nil")
	}

	db := m.db
	if tx != nil {
		db = tx
	}
	if db == nil {
		return errors.New("favorite venue reward record db is nil")
	}

	record.ActivityKey = strings.TrimSpace(record.ActivityKey)
	if record.ActivityKey == "" {
		return errors.New("activity key is required")
	}

	return db.Create(record).Error
}

func (m *FavoriteVenueRewardRecordModel) FindByActivityAndUser(activityKey string, userId int64) (*FavoriteVenueRewardRecord, error) {
	if m == nil || m.db == nil {
		return nil, errors.New("favorite venue reward record db is nil")
	}

	var record FavoriteVenueRewardRecord
	result := m.db.Where("activity_key = ? AND user_id = ?", strings.TrimSpace(activityKey), userId).Limit(1).Find(&record)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &record, nil
}

func (m *FavoriteVenueRewardRecordModel) FindByActivityAndVenue(activityKey string, venueId int64) (*FavoriteVenueRewardRecord, error) {
	if m == nil || m.db == nil {
		return nil, errors.New("favorite venue reward record db is nil")
	}

	var record FavoriteVenueRewardRecord
	result := m.db.Where("activity_key = ? AND venue_id = ?", strings.TrimSpace(activityKey), venueId).Limit(1).Find(&record)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &record, nil
}

func (m *FavoriteVenueRewardRecordModel) FindAdminList(activityKey string, page, pageSize int) ([]FavoriteVenueRewardRecordAdminItem, int64, error) {
	if m == nil || m.db == nil {
		return nil, 0, errors.New("favorite venue reward record db is nil")
	}

	page, pageSize = normalizePage(page, pageSize)
	offset := (page - 1) * pageSize
	activityKey = strings.TrimSpace(activityKey)

	query := m.db.Table("favorite_venue_reward_records AS records").
		Where("records.activity_key = ?", activityKey)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []FavoriteVenueRewardRecordAdminItem
	err := query.
		Select(`records.id,
			records.user_id,
			COALESCE(users.nickname, '') AS user_nickname,
			COALESCE(users.phone, '') AS user_phone,
			records.venue_id,
			COALESCE(venues.name, '') AS venue_name,
			records.reward_days,
			records.member_expires_at_before,
			records.member_expires_at_after,
			records.granted_at`).
		Joins("LEFT JOIN users ON users.id = records.user_id AND users.deleted_at IS NULL").
		Joins("LEFT JOIN venues ON venues.id = records.venue_id").
		Order("records.granted_at DESC, records.id DESC").
		Offset(offset).
		Limit(pageSize).
		Scan(&list).Error
	if err != nil {
		return nil, 0, err
	}

	return list, total, nil
}
