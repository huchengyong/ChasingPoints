package model

import (
	"strconv"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Season struct {
	Id             int64     `gorm:"primarykey"`
	Name           string    `gorm:"size:128;not null"`
	StartDate      time.Time `gorm:"not null;uniqueIndex:uk_seasons_start_date;index:idx_seasons_window,priority:1"`
	EndDate        time.Time `gorm:"not null;index:idx_seasons_window,priority:2"`
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
	Rewards        *string   `gorm:"type:json"`
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

func (m *SeasonModel) ListAllWithTx(tx *gorm.DB) ([]Season, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	var seasons []Season
	err := db.Order("start_date ASC, id ASC").Find(&seasons).Error
	return seasons, err
}

func (m *SeasonModel) FindByStartDate(startDate time.Time) (*Season, error) {
	return m.FindByStartDateWithTx(nil, startDate, false)
}

func (m *SeasonModel) FindByStartDateWithTx(tx *gorm.DB, startDate time.Time, lock bool) (*Season, error) {
	startDate = seasonDateOnly(startDate)
	db := m.db
	if tx != nil {
		db = tx
	}
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var season Season
	err := db.Where("start_date = ?", startDate).First(&season).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &season, err
}

// FindByStartDateCandidates reads at most two rows for one business-date
// window. The finite range also detects legacy rows written with a different
// time-zone offset for the same calendar start date.
func (m *SeasonModel) FindByStartDateCandidates(startDate time.Time) ([]Season, error) {
	return m.FindByStartDateCandidatesWithTx(nil, startDate, false)
}

func (m *SeasonModel) FindByStartDateCandidatesWithTx(tx *gorm.DB, startDate time.Time, lock bool) ([]Season, error) {
	startDate = seasonDateOnly(startDate)
	db := m.db
	if tx != nil {
		db = tx
	}
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var seasons []Season
	err := db.Where("start_date >= ? AND start_date < ?", startDate, startDate.AddDate(0, 0, 1)).
		Order("start_date ASC, id ASC").
		Limit(2).
		Find(&seasons).Error
	return seasons, err
}

func (m *SeasonModel) FindByEffectiveTime(at time.Time) (*Season, error) {
	return m.FindByEffectiveTimeWithTx(nil, at)
}

func (m *SeasonModel) FindByEffectiveTimeWithTx(tx *gorm.DB, at time.Time) (*Season, error) {
	return m.FindByEffectiveTimeInLocationWithTx(tx, at, nil)
}

func (m *SeasonModel) FindByEffectiveTimeInLocationWithTx(tx *gorm.DB, at time.Time, location *time.Location) (*Season, error) {
	day := seasonDateOnlyInLocation(at, location)
	db := m.db
	if tx != nil {
		db = tx
	}
	var season Season
	err := db.Where("start_date <= ? AND end_date >= ?", day, day).Order("start_date DESC, id DESC").First(&season).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &season, err
}

func (m *SeasonModel) FindNextAfterStartWithTx(tx *gorm.DB, startDate time.Time, lock bool) (*Season, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var season Season
	err := db.Where("start_date > ?", startDate).Order("start_date ASC, id ASC").First(&season).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &season, err
}

func (m *SeasonModel) CreateIfAbsentWithTx(tx *gorm.DB, season *Season) (bool, error) {
	if season == nil {
		return false, nil
	}
	db := m.db
	if tx != nil {
		db = tx
	}
	result := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "start_date"}},
		DoNothing: true,
	}).Create(season)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (m *SeasonModel) UpdateStatusToWithTx(tx *gorm.DB, seasonId int64, status int) (bool, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	result := db.Model(&Season{}).Where("id = ? AND status <> ?", seasonId, status).Update("status", status)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (m *SeasonModel) FindById(id int64) (*Season, error) {
	return m.FindByIdWithTx(nil, id, false)
}

func (m *SeasonModel) FindByIdWithTx(tx *gorm.DB, id int64, lock bool) (*Season, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var season Season
	err := db.First(&season, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &season, err
}

func (m *SeasonModel) FindCurrentWithTx(tx *gorm.DB, lock bool) (*Season, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var season Season
	err := db.Where("status = ?", 1).Order("start_date DESC, id DESC").First(&season).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &season, err
}

func (m *SeasonModel) FindReadyUpcomingWithTx(tx *gorm.DB, now time.Time, after time.Time, lock bool) (*Season, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	query := db.Where("status = ? AND start_date <= ?", 0, now)
	if !after.IsZero() {
		query = query.Where("start_date > ?", after)
	}
	var season Season
	err := query.Order("start_date ASC, id ASC").First(&season).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &season, err
}

func (m *SeasonModel) UpdateStatusWithTx(tx *gorm.DB, seasonId int64, fromStatus, toStatus int) (bool, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	result := db.Model(&Season{}).Where("id = ? AND status = ?", seasonId, fromStatus).Update("status", toStatus)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (m *SeasonModel) ListActive() ([]Season, error) {
	var seasons []Season
	err := m.db.Where("status = ?", 1).Order("start_date ASC, id ASC").Find(&seasons).Error
	return seasons, err
}

// FindDueUnsettledBatch returns a bounded, stable page of continuous-season
// windows that ended before dueBefore and have not published settlement.
func (m *SeasonModel) FindDueUnsettledBatch(startAt, dueBefore, afterEndDate time.Time, afterID int64, limit int) ([]Season, error) {
	startAt = seasonDateOnly(startAt)
	dueBefore = seasonDateOnly(dueBefore)
	if limit <= 0 {
		limit = 20
	}
	query := m.db.Table("seasons AS s").
		Select("s.*").
		Where("s.status IN ?", []int{0, 1}).
		Where("s.start_date >= ? AND s.end_date < ?", startAt, dueBefore).
		Where("NOT EXISTS (SELECT 1 FROM season_settlements AS ss WHERE ss.season_id = s.id AND ss.status = ?)", SeasonSettlementStatusCompleted)
	if !afterEndDate.IsZero() {
		query = query.Where("(s.end_date > ? OR (s.end_date = ? AND s.id > ?))", afterEndDate, afterEndDate, afterID)
	}
	var seasons []Season
	err := query.Order("s.end_date ASC, s.id ASC").Limit(limit).Find(&seasons).Error
	return seasons, err
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
	return m.ListAllWithTx(nil)
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

func (m *SeasonRecordModel) ListForSettlementWithTx(tx *gorm.DB, seasonId int64) ([]SeasonRecord, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	var records []SeasonRecord
	err := db.Where("season_id = ?", seasonId).
		Order("game_type ASC, end_rank_score DESC, wins DESC, id DESC").
		Find(&records).Error
	return records, err
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
	err := query.Order("end_rank_score DESC, wins DESC, id DESC").Offset(offset).Limit(pageSize).Find(&records).Error
	return records, total, err
}

type SeasonLeaderboardProfile struct {
	Id            int64  `gorm:"column:id"`
	UserId        int64  `gorm:"column:user_id"`
	EndRankScore  int    `gorm:"column:end_rank_score"`
	PeakRankScore int    `gorm:"column:peak_rank_score"`
	MatchesPlayed int    `gorm:"column:matches_played"`
	Wins          int    `gorm:"column:wins"`
	Nickname      string `gorm:"column:nickname"`
	Avatar        string `gorm:"column:avatar"`
}

func (m *SeasonRecordModel) FindLeaderboardWithProfilesByGameType(seasonId int64, gameType, page, pageSize int) ([]SeasonLeaderboardProfile, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	offset := (page - 1) * pageSize
	gameType = normalizeSeasonGameType(gameType)
	var total int64
	if err := m.db.Model(&SeasonRecord{}).Where("season_id = ? AND game_type = ?", seasonId, gameType).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var records []SeasonLeaderboardProfile
	err := m.db.Table("season_records AS r").
		Select("r.id, r.user_id, r.end_rank_score, r.peak_rank_score, r.matches_played, r.wins, u.nickname, u.avatar").
		Joins("LEFT JOIN users AS u ON u.id = r.user_id").
		Where("r.season_id = ? AND r.game_type = ?", seasonId, gameType).
		Order("r.end_rank_score DESC, r.wins DESC, r.id DESC").
		Offset(offset).
		Limit(pageSize).
		Scan(&records).Error
	return records, total, err
}

const defaultSeasonGameType = 3

func seasonDateOnly(value time.Time) time.Time {
	return seasonDateOnlyInLocation(value, nil)
}

func seasonDateOnlyInLocation(value time.Time, location *time.Location) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	if location != nil {
		value = value.In(location)
	}
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func normalizeSeasonGameType(gameType int) int {
	if gameType <= 0 {
		return defaultSeasonGameType
	}
	return gameType
}

type SeasonRecordMatchDelta struct {
	SeasonId       int64
	UserId         int64
	GameType       int
	StartRankScore int
	EndRankScore   int
	Won            bool
}

// ApplyCompetitiveMatchWithTx 以唯一赛季记录为门闩累加一场已结算排位赛。
func (m *SeasonRecordModel) ApplyCompetitiveMatchWithTx(tx *gorm.DB, delta SeasonRecordMatchDelta) error {
	db := m.db
	if tx != nil {
		db = tx
	}
	wins := 0
	if delta.Won {
		wins = 1
	}
	record := SeasonRecord{
		SeasonId:       delta.SeasonId,
		UserId:         delta.UserId,
		GameType:       normalizeSeasonGameType(delta.GameType),
		StartRankScore: delta.StartRankScore,
		EndRankScore:   delta.EndRankScore,
		PeakRankScore:  delta.EndRankScore,
		MatchesPlayed:  1,
		Wins:           wins,
	}
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "season_id"}, {Name: "user_id"}, {Name: "game_type"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"end_rank_score":  delta.EndRankScore,
			"peak_rank_score": gorm.Expr("CASE WHEN peak_rank_score >= ? THEN peak_rank_score ELSE ? END", delta.EndRankScore, delta.EndRankScore),
			"matches_played":  gorm.Expr("matches_played + 1"),
			"wins":            gorm.Expr("wins + ?", wins),
		}),
	}).Create(&record).Error
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
	return db.CreateInBatches(&records, 500).Error
}

func (m *SeasonRecordModel) UpsertBatchWithTx(tx *gorm.DB, records []SeasonRecord) error {
	if len(records) == 0 {
		return nil
	}
	db := m.db
	if tx != nil {
		db = tx
	}
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "season_id"},
			{Name: "user_id"},
			{Name: "game_type"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"start_rank_score",
			"end_rank_score",
			"peak_rank_score",
			"matches_played",
			"wins",
			"final_rank",
			"rewards",
		}),
	}).CreateInBatches(&records, 500).Error
}

func (m *SeasonRecordModel) DeleteAll(tx *gorm.DB) error {
	db := m.db
	if tx != nil {
		db = tx
	}
	return db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&SeasonRecord{}).Error
}

func (m *SeasonRecordModel) FindUserWinsBySeason(seasonId, userId int64) (map[string]int, error) {
	type row struct {
		GameType int
		Wins     int
	}
	var rows []row
	if err := m.db.Model(&SeasonRecord{}).
		Select("game_type, wins").
		Where("season_id = ? AND user_id = ?", seasonId, userId).
		Order("game_type ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]int, len(rows))
	for _, item := range rows {
		result[strconv.Itoa(item.GameType)] = item.Wins
	}
	return result, nil
}
