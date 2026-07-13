package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

var (
	ErrUserReputationProfileDBNil = errors.New("user reputation profile db is nil")
	ErrUserReputationProfileNil   = errors.New("user reputation profile is nil")
)

type UserReputationProfile struct {
	ID                      int64      `gorm:"primarykey" json:"id"`
	UserID                  int64      `gorm:"column:user_id;not null;uniqueIndex" json:"user_id"`
	ReputationScore         int        `gorm:"column:reputation_score;not null" json:"reputation_score"`
	LastRecoveredAt         *time.Time `gorm:"column:last_recovered_at" json:"last_recovered_at"`
	LastPenalizedAt         *time.Time `gorm:"column:last_penalized_at" json:"last_penalized_at"`
	BanUntil                *time.Time `gorm:"column:ban_until;index" json:"ban_until"`
	TotalPenaltyCount       int        `gorm:"column:total_penalty_count;not null;default:0" json:"total_penalty_count"`
	TotalAbnormalMatchCount int        `gorm:"column:total_abnormal_match_count;not null;default:0" json:"total_abnormal_match_count"`
	CreatedAt               time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt               time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (UserReputationProfile) TableName() string {
	return "user_reputation_profiles"
}

type UserReputationProfileModel struct {
	db *gorm.DB
}

func NewUserReputationProfileModel(db *gorm.DB) *UserReputationProfileModel {
	return &UserReputationProfileModel{db: db}
}

func (m *UserReputationProfileModel) FindByUserID(userID int64) (*UserReputationProfile, error) {
	return m.FindByUserIDWithTx(nil, userID)
}

func (m *UserReputationProfileModel) FindByUserIDWithTx(tx *gorm.DB, userID int64) (*UserReputationProfile, error) {
	if m == nil || m.db == nil {
		return nil, ErrUserReputationProfileDBNil
	}

	db := m.db
	if tx != nil {
		db = tx
	}

	var profile UserReputationProfile
	err := db.Where("user_id = ?", userID).First(&profile).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (m *UserReputationProfileModel) FindOrCreate(userID int64, initialScore int) (*UserReputationProfile, error) {
	return m.FindOrCreateWithTx(nil, userID, initialScore)
}

func (m *UserReputationProfileModel) FindOrCreateWithTx(tx *gorm.DB, userID int64, initialScore int) (*UserReputationProfile, error) {
	if m == nil {
		return nil, ErrUserReputationProfileDBNil
	}
	db := m.db
	if tx != nil {
		db = tx
	}
	if db == nil {
		return nil, ErrUserReputationProfileDBNil
	}

	profile := &UserReputationProfile{
		UserID:                  userID,
		ReputationScore:         initialScore,
		TotalPenaltyCount:       0,
		TotalAbnormalMatchCount: 0,
	}
	if err := db.Where("user_id = ?", userID).FirstOrCreate(profile).Error; err != nil {
		return nil, err
	}
	return profile, nil
}

func (m *UserReputationProfileModel) Save(profile *UserReputationProfile) error {
	return m.SaveWithTx(nil, profile)
}

func (m *UserReputationProfileModel) SaveWithTx(tx *gorm.DB, profile *UserReputationProfile) error {
	if m == nil {
		return ErrUserReputationProfileDBNil
	}
	db := m.db
	if tx != nil {
		db = tx
	}
	if db == nil {
		return ErrUserReputationProfileDBNil
	}
	if profile == nil {
		return ErrUserReputationProfileNil
	}
	return db.Save(profile).Error
}
