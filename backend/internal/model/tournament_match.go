package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type TournamentMatch struct {
	Id              int64          `gorm:"primarykey"`
	TournamentId    int64          `gorm:"not null;index"`
	SourceType      string         `gorm:"size:32;not null;default:''"`
	SourceMatchId   string         `gorm:"size:128;not null;default:'';index"`
	RoundName       string         `gorm:"size:128;not null;default:''"`
	RoundNumber     int            `gorm:"not null;default:0"`
	RoundOrder      int            `gorm:"not null;default:0;index"`
	MatchOrder      int            `gorm:"not null;default:0"`
	StartTime       *time.Time     `gorm:"default:null;index"`
	BestOf          int            `gorm:"not null;default:0"`
	HomePlayerId    int64          `gorm:"not null;default:0"`
	HomePlayerName  string         `gorm:"size:128;not null;default:''"`
	AwayPlayerId    int64          `gorm:"not null;default:0"`
	AwayPlayerName  string         `gorm:"size:128;not null;default:''"`
	HomeScore       int            `gorm:"not null;default:0"`
	AwayScore       int            `gorm:"not null;default:0"`
	WinnerSide      int            `gorm:"not null;default:0"`
	IsPlaceholder   bool           `gorm:"not null;default:false"`
	Player1Id       int64          `gorm:"not null;default:0"`
	Player2Id       int64          `gorm:"not null;default:0"`
	WinnerId        int64          `gorm:"not null;default:0"`
	MatchId         int64          `gorm:"not null;default:0"`
	BracketPosition string         `gorm:"size:32;not null;default:''"`
	Status          int            `gorm:"not null;default:0;index"`
	CreatedAt       time.Time      `gorm:"autoCreateTime"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

func (TournamentMatch) TableName() string {
	return "tournament_matches"
}

type TournamentMatchModel struct {
	db *gorm.DB
}

func NewTournamentMatchModel(db *gorm.DB) *TournamentMatchModel {
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
		Order("round_order ASC, round_number ASC, match_order ASC, id ASC").
		Find(&list).Error
	return list, err
}

func (m *TournamentMatchModel) FindByTournamentIds(tournamentIds []int64) (map[int64][]TournamentMatch, error) {
	result := make(map[int64][]TournamentMatch, len(tournamentIds))
	if len(tournamentIds) == 0 {
		return result, nil
	}

	var list []TournamentMatch
	if err := m.db.Where("tournament_id IN ?", tournamentIds).
		Order("round_order ASC, round_number ASC, match_order ASC, id ASC").
		Find(&list).Error; err != nil {
		return nil, err
	}

	for _, tournamentId := range tournamentIds {
		result[tournamentId] = []TournamentMatch{}
	}
	for _, item := range list {
		result[item.TournamentId] = append(result[item.TournamentId], item)
	}
	return result, nil
}

func (m *TournamentMatchModel) FindBySourceMatchIds(sourceType string, sourceMatchIds []string) (map[string]TournamentMatch, error) {
	result := make(map[string]TournamentMatch, len(sourceMatchIds))
	if len(sourceMatchIds) == 0 {
		return result, nil
	}

	var list []TournamentMatch
	if err := m.db.
		Where("source_type = ? AND source_match_id IN ?", sourceType, sourceMatchIds).
		Find(&list).Error; err != nil {
		return nil, err
	}

	for _, item := range list {
		result[item.SourceMatchId] = item
	}
	return result, nil
}

func (m *TournamentMatchModel) Update(match *TournamentMatch) error {
	return m.db.Save(match).Error
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

func (m *TournamentMatchModel) DeleteById(id int64) error {
	return m.db.Where("id = ?", id).Delete(&TournamentMatch{}).Error
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
