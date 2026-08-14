package model

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

type ChallengeProfile struct {
	Id           int64     `gorm:"column:id"`
	FromUserId   int64     `gorm:"column:from_user_id"`
	ToUserId     int64     `gorm:"column:to_user_id"`
	GameType     int       `gorm:"column:game_type"`
	Message      string    `gorm:"column:message"`
	Status       int       `gorm:"column:status"`
	MatchId      *int64    `gorm:"column:match_id"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	FromNickname string    `gorm:"column:from_nickname"`
	FromAvatar   string    `gorm:"column:from_avatar"`
	ToNickname   string    `gorm:"column:to_nickname"`
	ToAvatar     string    `gorm:"column:to_avatar"`
}

const pendingChallengeReadLimit = 200

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

func (m *ChallengeModel) FindByIdForUpdateWithTx(tx *gorm.DB, id int64) (*Challenge, error) {
	var challenge Challenge
	db := m.db
	if tx != nil {
		db = tx
	}
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&challenge, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &challenge, err
}

func (m *ChallengeModel) GetPendingByUserId(userId int64) ([]Challenge, error) {
	now := time.Now()
	var list []Challenge
	err := m.db.Where("(to_user_id = ? OR from_user_id = ?) AND ((status = 0 AND expires_at > ?) OR status IN (1, 2, 3))", userId, userId, now).
		Order("created_at DESC, id DESC").
		Limit(pendingChallengeReadLimit).
		Find(&list).Error
	return list, err
}

func (m *ChallengeModel) GetPendingByUserIdWithProfiles(userId int64) ([]ChallengeProfile, error) {
	var list []ChallengeProfile
	now := time.Now()
	err := m.db.Table("challenges AS c").
		Select(`c.id, c.from_user_id, c.to_user_id, c.game_type, c.message, c.status, c.match_id, c.created_at,
			f.nickname AS from_nickname, f.avatar AS from_avatar, t.nickname AS to_nickname, t.avatar AS to_avatar`).
		Joins("LEFT JOIN users AS f ON f.id = c.from_user_id").
		Joins("LEFT JOIN users AS t ON t.id = c.to_user_id").
		Where("(c.to_user_id = ? OR c.from_user_id = ?) AND ((c.status = 0 AND c.expires_at > ?) OR c.status IN (1, 2, 3))", userId, userId, now).
		Order("c.created_at DESC, c.id DESC").
		Limit(pendingChallengeReadLimit).
		Scan(&list).Error
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
		Where("id = ? AND to_user_id = ? AND status = 0 AND expires_at > ?", challengeId, userId, time.Now()).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (m *ChallengeModel) LinkAcceptedWithTx(tx *gorm.DB, challengeId, fromUserId, toUserId int64, gameType int, matchId int64) error {
	db := m.db
	if tx != nil {
		db = tx
	}
	result := db.Model(&Challenge{}).
		Where("id = ? AND status = 1 AND match_id IS NULL AND game_type = ? AND ((from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?))",
			challengeId, gameType, fromUserId, toUserId, toUserId, fromUserId).
		Updates(map[string]interface{}{"match_id": matchId})
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
		Where("id = ? AND to_user_id = ? AND status = 0 AND expires_at > ?", challengeId, userId, time.Now()).
		Update("status", 2)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (m *ChallengeModel) ListExpiredPending(limit int, now time.Time) ([]Challenge, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	var list []Challenge
	err := m.db.Where("status = 0 AND expires_at <= ?", now).
		Order("expires_at ASC, id ASC").
		Limit(limit).
		Find(&list).Error
	return list, err
}

func (m *ChallengeModel) MarkExpiredIfPending(challengeID int64, now time.Time) (bool, error) {
	result := m.db.Model(&Challenge{}).
		Where("id = ? AND status = 0 AND expires_at <= ?", challengeID, now).
		Update("status", 3)
	return result.RowsAffected > 0, result.Error
}
