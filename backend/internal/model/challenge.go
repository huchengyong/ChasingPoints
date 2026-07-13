package model

import (
	"time"

	"gorm.io/gorm"
)

type Challenge struct {
	Id         int64     `gorm:"primarykey"`
	FromUserId int64     `gorm:"not null;index"`
	ToUserId   int64     `gorm:"not null;index"`
	GameType   int       `gorm:"not null"`
	Message    string    `gorm:"size:200"`
	Status     int       `gorm:"not null;default:0"`
	MatchId    *int64    `gorm:"index"`
	ExpiresAt  time.Time `gorm:"not null"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

func (Challenge) TableName() string {
	return "challenges"
}

type ChallengeModel struct {
	db *gorm.DB
}

func NewChallengeModel(db *gorm.DB) *ChallengeModel {
	return &ChallengeModel{db: db}
}

func (m *ChallengeModel) Create(challenge *Challenge) error {
	return m.db.Create(challenge).Error
}

func (m *ChallengeModel) FindById(id int64) (*Challenge, error) {
	var challenge Challenge
	err := m.db.First(&challenge, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &challenge, err
}

func (m *ChallengeModel) GetPendingByUserId(userId int64) ([]Challenge, error) {
	now := time.Now()
	var list []Challenge
	err := m.db.Where("(to_user_id = ? OR from_user_id = ?) AND status = 0 AND expires_at > ?", userId, userId, now).
		Order("created_at DESC, id DESC").
		Find(&list).Error
	return list, err
}

func (m *ChallengeModel) Accept(challengeId, userId, matchId int64) error {
	updates := map[string]interface{}{
		"status": 1,
	}
	if matchId > 0 {
		updates["match_id"] = matchId
	} else {
		updates["match_id"] = nil
	}

	result := m.db.Model(&Challenge{}).
		Where("id = ? AND to_user_id = ? AND status = 0", challengeId, userId).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (m *ChallengeModel) Reject(challengeId, userId int64) error {
	result := m.db.Model(&Challenge{}).
		Where("id = ? AND to_user_id = ? AND status = 0", challengeId, userId).
		Update("status", 2)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (m *ChallengeModel) ExpireOld() error {
	now := time.Now()
	return m.db.Model(&Challenge{}).
		Where("status = 0 AND expires_at < ?", now).
		Update("status", 3).Error
}
