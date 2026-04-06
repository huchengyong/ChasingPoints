package model

import (
	"time"

	"gorm.io/gorm"
)

type Player struct {
	Id             int64          `gorm:"primarykey"`
	SourceType     string         `gorm:"size:32;not null;default:''"`
	SourcePlayerId string         `gorm:"size:128;not null;default:'';index"`
	FirstName      string         `gorm:"size:64;not null;default:''"`
	LastName       string         `gorm:"size:64;not null;default:''"`
	DisplayName    string         `gorm:"size:128;not null;default:''"`
	Avatar         string         `gorm:"size:512;not null;default:''"`
	CountryCode    string         `gorm:"size:32;not null;default:''"`
	FlagEmoji      string         `gorm:"size:16;not null;default:''"`
	CreatedAt      time.Time      `gorm:"autoCreateTime"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

func (Player) TableName() string {
	return "players"
}

type PlayerModel struct {
	db *gorm.DB
}

func NewPlayerModel(db *gorm.DB) *PlayerModel {
	return &PlayerModel{db: db}
}

func (m *PlayerModel) Create(player *Player) error {
	return m.db.Create(player).Error
}

func (m *PlayerModel) FindByIds(ids []int64) (map[int64]Player, error) {
	result := make(map[int64]Player, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	var list []Player
	if err := m.db.Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}

	for _, item := range list {
		result[item.Id] = item
	}
	return result, nil
}

func (m *PlayerModel) FindBySourcePlayerIds(sourceType string, sourcePlayerIds []string) (map[string]Player, error) {
	result := make(map[string]Player, len(sourcePlayerIds))
	if len(sourcePlayerIds) == 0 {
		return result, nil
	}

	var list []Player
	if err := m.db.
		Where("source_type = ? AND source_player_id IN ?", sourceType, sourcePlayerIds).
		Find(&list).Error; err != nil {
		return nil, err
	}

	for _, item := range list {
		result[item.SourcePlayerId] = item
	}
	return result, nil
}
