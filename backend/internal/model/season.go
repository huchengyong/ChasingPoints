package model

import (
	"strconv"
	"time"

	"gorm.io/gorm"
)

type Season struct {
	Id             int64     `gorm:"primarykey"`
	Name           string    `gorm:"size:128;not null"`
	StartDate      time.Time `gorm:"not null;index"`
	EndDate        time.Time `gorm:"not null"`
	Status         int       `gorm:"not null;default:0;index"`
	RankResetRatio float64   `gorm:"not null;default:0.7"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
}

func (Season) TableName() string {
	return "seasons"
}

type SeasonRecord struct {
	Id             int64     `gorm:"primarykey"`
	SeasonId       int64     `gorm:"not null;index;uniqueIndex:uk_season_user_game"`
	UserId         int64     `gorm:"not null;index;uniqueIndex:uk_season_user_game"`
	GameType       int       `gorm:"not null;default:3;index;uniqueIndex:uk_season_user_game" json:"game_type"`
	StartRankScore int       `gorm:"not null;default:0"`
	EndRankScore   int       `gorm:"not null;default:0"`
	PeakRankScore  int       `gorm:"not null;default:0"`
	MatchesPlayed  int       `gorm:"not null;default:0"`
	Wins           int       `gorm:"not null;default:0"`
	FinalRank      int       `gorm:"not null;default:0"`
	Rewards        string    `gorm:"type:text"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
}

func (SeasonRecord) TableName() string {
	return "season_records"
}

type SeasonModel struct {
	db *gorm.DB
}

func NewSeasonModel(db *gorm.DB) *SeasonModel {
	return &SeasonModel{db: db}
}

func (m *SeasonModel) FindCurrent() (*Season, error) {
	var season Season
	err := m.db.Where("status = ?", 1).Order("start_date DESC, id DESC").First(&season).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &season, err
}

func (m *SeasonModel) FindById(id int64) (*Season, error) {
	var season Season
	err := m.db.First(&season, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &season, err
}

func (m *SeasonModel) FindLatest() (*Season, error) {
	var season Season
	err := m.db.Order("end_date DESC, id DESC").First(&season).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &season, err
}

func (m *SeasonModel) ListAll() ([]Season, error) {
	var seasons []Season
	err := m.db.Order("start_date ASC, id ASC").Find(&seasons).Error
	return seasons, err
}

func (m *SeasonModel) Create(season *Season) error {
	return m.db.Create(season).Error
}

type SeasonRecordModel struct {
	db *gorm.DB
}

func NewSeasonRecordModel(db *gorm.DB) *SeasonRecordModel {
	return &SeasonRecordModel{db: db}
}

func (m *SeasonRecordModel) FindBySeasonAndUser(seasonId, userId int64) (*SeasonRecord, error) {
	return m.FindBySeasonAndUserAndGameType(seasonId, userId, defaultSeasonGameType)
}

func (m *SeasonRecordModel) FindBySeasonAndUserAndGameType(seasonId, userId int64, gameType int) (*SeasonRecord, error) {
	var record SeasonRecord
	err := m.db.Where("season_id = ? AND user_id = ? AND game_type = ?", seasonId, userId, normalizeSeasonGameType(gameType)).First(&record).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &record, err
}

func (m *SeasonRecordModel) FindLeaderboard(seasonId int64, page, pageSize int) ([]SeasonRecord, int64, error) {
	return m.FindLeaderboardByGameType(seasonId, defaultSeasonGameType, page, pageSize)
}

func (m *SeasonRecordModel) FindLeaderboardByGameType(seasonId int64, gameType, page, pageSize int) ([]SeasonRecord, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	offset := (page - 1) * pageSize

	query := m.db.Model(&SeasonRecord{}).Where("season_id = ? AND game_type = ?", seasonId, normalizeSeasonGameType(gameType))

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var records []SeasonRecord
	err := query.Order("end_rank_score DESC, wins DESC, id ASC").Offset(offset).Limit(pageSize).Find(&records).Error
	return records, total, err
}

const defaultSeasonGameType = 3

func normalizeSeasonGameType(gameType int) int {
	if gameType <= 0 {
		return defaultSeasonGameType
	}
	return gameType
}

func (m *SeasonRecordModel) Create(record *SeasonRecord) error {
	return m.db.Create(record).Error
}

func (m *SeasonRecordModel) CreateBatch(records []SeasonRecord) error {
	return m.CreateBatchWithTx(nil, records)
}

func (m *SeasonRecordModel) CreateBatchWithTx(tx *gorm.DB, records []SeasonRecord) error {
	if len(records) == 0 {
		return nil
	}
	db := m.db
	if tx != nil {
		db = tx
	}
	return db.Create(&records).Error
}

func (m *SeasonRecordModel) DeleteAll(tx *gorm.DB) error {
	db := m.db
	if tx != nil {
		db = tx
	}
	return db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&SeasonRecord{}).Error
}

func (m *SeasonRecordModel) FindUserWinByTypeInSeason(userId int64, startDate, endDate time.Time) (map[string]int, error) {
	type row struct {
		GameType int
		Wins     int
	}

	var rows []row
	err := m.db.Table("matches").
		Select("game_type, SUM(CASE WHEN result = 1 THEN 1 ELSE 0 END) AS wins").
		Where("user_id = ? AND status = 2 AND match_time >= ? AND match_time <= ?", userId, startDate, endDate).
		Group("game_type").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]int, len(rows))
	for _, item := range rows {
		result[strconv.Itoa(item.GameType)] = item.Wins
	}

	return result, nil
}
