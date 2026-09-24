package model

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	SeasonSettlementStatusRunning   = "running"
	SeasonSettlementStatusCompleted = "completed"
	SeasonSettlementStatusFailed    = "failed"
)

type SeasonChallengeSnapshot struct {
	Id            int64     `gorm:"primarykey"`
	SeasonId      int64     `gorm:"not null;index:idx_season_challenge_snapshots_user_season,priority:2;index:idx_season_challenge_snapshots_season,priority:1;uniqueIndex:uk_season_challenge_snapshot,priority:1"`
	UserId        int64     `gorm:"not null;index:idx_season_challenge_snapshots_user_season,priority:1;uniqueIndex:uk_season_challenge_snapshot,priority:2"`
	GameType      int       `gorm:"not null;index:idx_season_challenge_snapshots_user_season,priority:3;index:idx_season_challenge_snapshots_season,priority:2;uniqueIndex:uk_season_challenge_snapshot,priority:3"`
	ChallengeKey  string    `gorm:"size:64;not null;uniqueIndex:uk_season_challenge_snapshot,priority:4"`
	ChallengeName string    `gorm:"size:128;not null"`
	Threshold     int       `gorm:"not null;default:0"`
	Progress      int       `gorm:"not null;default:0"`
	Completed     int       `gorm:"not null;default:0"`
	ArchivedAt    time.Time `gorm:"not null"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
}

func (SeasonChallengeSnapshot) TableName() string {
	return "season_challenge_snapshots"
}

type SeasonChallengeSnapshotModel struct {
	db *gorm.DB
}

func NewSeasonChallengeSnapshotModel(db *gorm.DB) *SeasonChallengeSnapshotModel {
	return &SeasonChallengeSnapshotModel{db: db}
}

func (m *SeasonChallengeSnapshotModel) UpsertBatchWithTx(tx *gorm.DB, snapshots []SeasonChallengeSnapshot) error {
	if len(snapshots) == 0 {
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
			{Name: "challenge_key"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"challenge_name",
			"threshold",
			"progress",
			"completed",
			"archived_at",
		}),
	}).Create(&snapshots).Error
}

func (m *SeasonChallengeSnapshotModel) FindByUserSeasonAndGameType(userId, seasonId int64, gameType int) ([]SeasonChallengeSnapshot, error) {
	var list []SeasonChallengeSnapshot
	err := m.db.Where("user_id = ? AND season_id = ? AND game_type = ?", userId, seasonId, gameType).
		Order("id ASC").
		Find(&list).Error
	return list, err
}

type SeasonSettlement struct {
	Id           int64  `gorm:"primarykey"`
	SeasonId     int64  `gorm:"not null;uniqueIndex:uk_season_settlements_season"`
	NextSeasonId *int64 `gorm:"type:bigint unsigned"`
	Status       string `gorm:"size:16;not null;default:'running';index:idx_season_settlements_status,priority:1"`
	Attempts     int    `gorm:"not null;default:0"`
	StartedAt    *time.Time
	CompletedAt  *time.Time
	LastError    string    `gorm:"size:512;not null;default:''"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime;index:idx_season_settlements_status,priority:2"`
}

func (SeasonSettlement) TableName() string {
	return "season_settlements"
}

type SeasonSettlementModel struct {
	db *gorm.DB
}

func NewSeasonSettlementModel(db *gorm.DB) *SeasonSettlementModel {
	return &SeasonSettlementModel{db: db}
}

func (m *SeasonSettlementModel) FindBySeasonId(seasonId int64) (*SeasonSettlement, error) {
	return m.FindBySeasonIdWithTx(nil, seasonId, false)
}

func (m *SeasonSettlementModel) FindBySeasonIdWithTx(tx *gorm.DB, seasonId int64, lock bool) (*SeasonSettlement, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var settlement SeasonSettlement
	err := db.Where("season_id = ?", seasonId).First(&settlement).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &settlement, err
}

func (m *SeasonSettlementModel) CreateWithTx(tx *gorm.DB, settlement *SeasonSettlement) error {
	db := m.db
	if tx != nil {
		db = tx
	}
	return db.Create(settlement).Error
}

func (m *SeasonSettlementModel) FindOrCreateForUpdateWithTx(tx *gorm.DB, seasonId int64) (*SeasonSettlement, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "season_id"}},
		DoNothing: true,
	}).Create(&SeasonSettlement{
		SeasonId: seasonId,
		Status:   SeasonSettlementStatusRunning,
	}).Error; err != nil {
		return nil, err
	}
	return m.FindBySeasonIdWithTx(tx, seasonId, true)
}

func (m *SeasonSettlementModel) SaveWithTx(tx *gorm.DB, settlement *SeasonSettlement) error {
	db := m.db
	if tx != nil {
		db = tx
	}
	return db.Save(settlement).Error
}
