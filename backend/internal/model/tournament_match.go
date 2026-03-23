package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type TournamentMatch struct {
	Id              int64     `gorm:"primarykey"`
	TournamentId    int64     `gorm:"not null;index"`
	RoundNumber     int       `gorm:"not null"`
	MatchOrder      int       `gorm:"not null"`
	Player1Id       int64     `gorm:"not null;default:0"`
	Player2Id       int64     `gorm:"not null;default:0"`
	WinnerId        int64     `gorm:"not null;default:0"`
	MatchId         int64     `gorm:"default:null"`
	BracketPosition string    `gorm:"size:32;not null;default:''"`
	Status          int       `gorm:"not null;default:0"`
	CreatedAt       time.Time `gorm:"autoCreateTime"`
}

func (TournamentMatch) TableName() string {
	return "tournament_matches"
}

type TournamentMatchModel struct {
	db *gorm.DB
}

func NewTournamentMatchModel(db *gorm.DB) *TournamentMatchModel {
	_ = db.AutoMigrate(&TournamentMatch{})
	return &TournamentMatchModel{db: db}
}

func (m *TournamentMatchModel) Create(match *TournamentMatch) error {
	return m.db.Create(match).Error
}

func (m *TournamentMatchModel) CreateBatch(matches []TournamentMatch) error {
	if len(matches) == 0 {
		return nil
	}
	return m.db.Create(&matches).Error
}

func (m *TournamentMatchModel) FindById(id int64) (*TournamentMatch, error) {
	var match TournamentMatch
	err := m.db.First(&match, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &match, err
}

func (m *TournamentMatchModel) FindByTournament(tournamentId int64) ([]TournamentMatch, error) {
	var list []TournamentMatch
	err := m.db.Where("tournament_id = ?", tournamentId).
		Order("round_number ASC, match_order ASC, id ASC").
		Find(&list).Error
	return list, err
}

func (m *TournamentMatchModel) UpdateWinner(id, winnerId int64, status int) error {
	return m.db.Model(&TournamentMatch{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"winner_id": winnerId,
			"status":    status,
		}).Error
}

func (m *TournamentMatchModel) UpdateMatchId(id, matchId int64) error {
	return m.db.Model(&TournamentMatch{}).
		Where("id = ?", id).
		Update("match_id", matchId).Error
}

func (m *TournamentMatchModel) UpdatePlayer(id int64, playerField string, playerId int64) error {
	if playerField != "player1_id" && playerField != "player2_id" {
		return errors.New("invalid player field")
	}

	return m.db.Model(&TournamentMatch{}).
		Where("id = ?", id).
		Update(playerField, playerId).Error
}

func (m *TournamentMatchModel) DeleteByTournament(tournamentId int64) error {
	return m.db.Where("tournament_id = ?", tournamentId).Delete(&TournamentMatch{}).Error
}

func (m *TournamentParticipantModel) UpdateFinalRankAndStatus(tournamentId, userId int64, finalRank, status int) (bool, error) {
	result := m.db.Model(&TournamentParticipant{}).
		Where("tournament_id = ? AND user_id = ?", tournamentId, userId).
		Updates(map[string]interface{}{
			"final_rank": finalRank,
			"status":     status,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}
