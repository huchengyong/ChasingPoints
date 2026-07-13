package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

var ErrUserNotificationPreferenceDBNil = errors.New("user notification preference db is nil")

type UserNotificationPreference struct {
	Id                   int64     `gorm:"primarykey"`
	UserId               int64     `gorm:"not null;uniqueIndex"`
	MatchResultEnabled   bool      `gorm:"not null;default:true"`
	FriendRequestEnabled bool      `gorm:"not null;default:true"`
	ChallengeEnabled     bool      `gorm:"not null;default:true"`
	TournamentEnabled    bool      `gorm:"not null;default:true"`
	FollowEnabled        bool      `gorm:"not null;default:true"`
	CreatedAt            time.Time `gorm:"autoCreateTime"`
	UpdatedAt            time.Time `gorm:"autoUpdateTime"`
}

func (UserNotificationPreference) TableName() string {
	return "user_notification_preferences"
}

func DefaultUserNotificationPreference(userId int64) *UserNotificationPreference {
	return &UserNotificationPreference{
		UserId:               userId,
		MatchResultEnabled:   true,
		FriendRequestEnabled: true,
		ChallengeEnabled:     true,
		TournamentEnabled:    true,
		FollowEnabled:        true,
	}
}

type UserNotificationPreferenceModel struct {
	db *gorm.DB
}

func NewUserNotificationPreferenceModel(db *gorm.DB) *UserNotificationPreferenceModel {
	return &UserNotificationPreferenceModel{db: db}
}

func (m *UserNotificationPreferenceModel) FindByUserId(userId int64) (*UserNotificationPreference, error) {
	if m == nil || m.db == nil {
		return nil, ErrUserNotificationPreferenceDBNil
	}

	var prefs UserNotificationPreference
	err := m.db.Where("user_id = ?", userId).First(&prefs).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &prefs, nil
}

func (m *UserNotificationPreferenceModel) GetByUserIdOrDefault(userId int64) (*UserNotificationPreference, error) {
	if m == nil || m.db == nil {
		return nil, ErrUserNotificationPreferenceDBNil
	}

	prefs, err := m.FindByUserId(userId)
	if err != nil {
		return nil, err
	}
	if prefs != nil {
		return prefs, nil
	}
	return DefaultUserNotificationPreference(userId), nil
}

func (m *UserNotificationPreferenceModel) Upsert(prefs *UserNotificationPreference) error {
	if m == nil || m.db == nil {
		return ErrUserNotificationPreferenceDBNil
	}
	if prefs == nil {
		return errors.New("user notification preference is nil")
	}

	existing, err := m.FindByUserId(prefs.UserId)
	if err != nil {
		return err
	}
	if existing == nil {
		return m.db.Model(&UserNotificationPreference{}).Create(map[string]interface{}{
			"user_id":                prefs.UserId,
			"match_result_enabled":   prefs.MatchResultEnabled,
			"friend_request_enabled": prefs.FriendRequestEnabled,
			"challenge_enabled":      prefs.ChallengeEnabled,
			"tournament_enabled":     prefs.TournamentEnabled,
			"follow_enabled":         prefs.FollowEnabled,
		}).Error
	}

	return m.db.Model(&UserNotificationPreference{}).
		Where("id = ?", existing.Id).
		Updates(map[string]interface{}{
			"match_result_enabled":   prefs.MatchResultEnabled,
			"friend_request_enabled": prefs.FriendRequestEnabled,
			"challenge_enabled":      prefs.ChallengeEnabled,
			"tournament_enabled":     prefs.TournamentEnabled,
			"follow_enabled":         prefs.FollowEnabled,
		}).Error
}
