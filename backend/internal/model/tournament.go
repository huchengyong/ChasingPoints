package model

import (
	"time"

	"gorm.io/gorm"
)

type Tournament struct {
	Id             int64      `gorm:"primarykey"`
	CreatorId      int64      `gorm:"not null;index"`
	Name           string     `gorm:"size:128;not null"`
	Description    string     `gorm:"type:text"`
	GameType       int        `gorm:"not null;index"`
	Format         int        `gorm:"not null;default:1"`
	MaxPlayers     int        `gorm:"not null;default:16"`
	CurrentPlayers int        `gorm:"not null;default:0"`
	Status         int        `gorm:"not null;default:0;index"`
	City           string     `gorm:"size:64;not null;default:'';index"`
	VenueName      string     `gorm:"size:128;not null;default:''"`
	StartTime      *time.Time `gorm:"default:null;index"`
	EndTime        *time.Time `gorm:"default:null"`
	CreatedAt      time.Time  `gorm:"autoCreateTime"`
}

func (Tournament) TableName() string {
	return "tournaments"
}

type TournamentParticipant struct {
	Id           int64     `gorm:"primarykey"`
	TournamentId int64     `gorm:"not null;uniqueIndex:uk_tournament_user;index"`
	UserId       int64     `gorm:"not null;uniqueIndex:uk_tournament_user;index"`
	Seed         int       `gorm:"not null;default:0"`
	Status       int       `gorm:"not null;default:0"`
	FinalRank    int       `gorm:"not null;default:0"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
}

func (TournamentParticipant) TableName() string {
	return "tournament_participants"
}

type TournamentModel struct {
	db *gorm.DB
}

func NewTournamentModel(db *gorm.DB) *TournamentModel {
	return &TournamentModel{db: db}
}

func (m *TournamentModel) Create(tournament *Tournament) error {
	return m.db.Create(tournament).Error
}

func (m *TournamentModel) Update(tournament *Tournament) error {
	return m.db.Save(tournament).Error
}

func (m *TournamentModel) FindById(id int64) (*Tournament, error) {
	var tournament Tournament
	err := m.db.First(&tournament, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &tournament, err
}

func (m *TournamentModel) FindByIdWithDB(db *gorm.DB, id int64) (*Tournament, error) {
	var tournament Tournament
	err := db.First(&tournament, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &tournament, err
}

func (m *TournamentModel) FindByIds(ids []int64) (map[int64]Tournament, error) {
	result := make(map[int64]Tournament, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	var list []Tournament
	if err := m.db.Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}

	for _, item := range list {
		result[item.Id] = item
	}
	return result, nil
}

func (m *TournamentModel) FindList(page, pageSize int, city string, gameType, status int, useStatusFilter bool) ([]Tournament, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	offset := (page - 1) * pageSize

	query := m.db.Model(&Tournament{})
	if city != "" {
		query = query.Where("city = ?", city)
	}
	if gameType > 0 {
		query = query.Where("game_type = ?", gameType)
	}
	if useStatusFilter {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []Tournament
	err := query.Order("start_time ASC, id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (m *TournamentModel) FindListByParticipant(userId int64, page, pageSize, status int, useStatusFilter bool) ([]Tournament, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	offset := (page - 1) * pageSize

	query := m.db.Model(&Tournament{}).
		Joins("JOIN tournament_participants tp ON tp.tournament_id = tournaments.id").
		Where("tp.user_id = ?", userId)
	if useStatusFilter {
		query = query.Where("tournaments.status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []Tournament
	err := query.Order("tournaments.start_time DESC, tournaments.id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (m *TournamentModel) UpdateStatusByCreator(tournamentId, creatorId int64, status int) (bool, error) {
	result := m.db.Model(&Tournament{}).
		Where("id = ? AND creator_id = ?", tournamentId, creatorId).
		Update("status", status)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (m *TournamentModel) IncrementCurrentPlayersWithDB(db *gorm.DB, tournamentId int64) (bool, error) {
	result := db.Model(&Tournament{}).
		Where("id = ? AND current_players < max_players AND status = 0", tournamentId).
		Update("current_players", gorm.Expr("current_players + 1"))
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (m *TournamentModel) DecrementCurrentPlayersWithDB(db *gorm.DB, tournamentId int64) (bool, error) {
	result := db.Model(&Tournament{}).
		Where("id = ? AND current_players > 0", tournamentId).
		Update("current_players", gorm.Expr("current_players - 1"))
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (m *TournamentModel) Transaction(fc func(tx *gorm.DB) error) error {
	return m.db.Transaction(fc)
}

func (m *TournamentModel) DeleteById(id int64) error {
	return m.db.Where("id = ?", id).Delete(&Tournament{}).Error
}

type TournamentParticipantModel struct {
	db *gorm.DB
}

func NewTournamentParticipantModel(db *gorm.DB) *TournamentParticipantModel {
	return &TournamentParticipantModel{db: db}
}

func (m *TournamentParticipantModel) Create(participant *TournamentParticipant) error {
	return m.db.Create(participant).Error
}

func (m *TournamentParticipantModel) CreateWithDB(db *gorm.DB, participant *TournamentParticipant) error {
	return db.Create(participant).Error
}

func (m *TournamentParticipantModel) FindByTournamentAndUser(tournamentId, userId int64) (*TournamentParticipant, error) {
	var participant TournamentParticipant
	err := m.db.Where("tournament_id = ? AND user_id = ?", tournamentId, userId).First(&participant).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &participant, err
}

func (m *TournamentParticipantModel) FindByTournamentAndUserWithDB(db *gorm.DB, tournamentId, userId int64) (*TournamentParticipant, error) {
	var participant TournamentParticipant
	err := db.Where("tournament_id = ? AND user_id = ?", tournamentId, userId).First(&participant).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &participant, err
}

func (m *TournamentParticipantModel) FindListByTournamentId(tournamentId int64) ([]TournamentParticipant, error) {
	var list []TournamentParticipant
	err := m.db.Where("tournament_id = ?", tournamentId).
		Order("seed ASC, created_at ASC, id ASC").
		Find(&list).Error
	return list, err
}

func (m *TournamentParticipantModel) DeleteByTournamentAndUserWithDB(db *gorm.DB, tournamentId, userId int64) (bool, error) {
	result := db.Where("tournament_id = ? AND user_id = ?", tournamentId, userId).Delete(&TournamentParticipant{})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (m *TournamentParticipantModel) UpdateStatusByTournamentAndUser(tournamentId, userId int64, status int) (bool, error) {
	result := m.db.Model(&TournamentParticipant{}).
		Where("tournament_id = ? AND user_id = ?", tournamentId, userId).
		Update("status", status)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
