package model

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

const MemberGrowthSourceRealMatchCompleted = "real_match_completed"

var (
	ErrMemberGrowthProfileDBNil = errors.New("member growth profile db is nil")
	ErrMemberGrowthProfileNil   = errors.New("member growth profile is nil")
	ErrMemberGrowthLogDBNil     = errors.New("member growth log db is nil")
	ErrMemberGrowthLogNil       = errors.New("member growth log is nil")
)

type MemberGrowthProfile struct {
	Id               int64      `gorm:"primarykey" json:"id"`
	UserId           int64      `gorm:"not null;uniqueIndex" json:"user_id"`
	GrowthPoints     int        `gorm:"not null;default:0" json:"growth_points"`
	GrowthLevel      int        `gorm:"not null;default:1" json:"growth_level"`
	TodayGrowthCount int        `gorm:"not null;default:0" json:"today_growth_count"`
	TodayGrowthDate  *time.Time `json:"today_growth_date"`
	LastGrowthAt     *time.Time `json:"last_growth_at"`
	CreatedAt        time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (MemberGrowthProfile) TableName() string {
	return "member_growth_profiles"
}

type MemberGrowthLog struct {
	Id           int64     `gorm:"primarykey" json:"id"`
	UserId       int64     `gorm:"not null;uniqueIndex:uniq_member_growth_log,priority:1;index" json:"user_id"`
	MatchId      int64     `gorm:"not null;uniqueIndex:uniq_member_growth_log,priority:2" json:"match_id"`
	GrowthPoints int       `gorm:"not null;default:1" json:"growth_points"`
	Source       string    `gorm:"size:32;not null;uniqueIndex:uniq_member_growth_log,priority:3" json:"source"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (MemberGrowthLog) TableName() string {
	return "member_growth_logs"
}

type MemberGrowthProfileModel struct {
	db *gorm.DB
}

func NewMemberGrowthProfileModel(db *gorm.DB) *MemberGrowthProfileModel {
	return &MemberGrowthProfileModel{db: db}
}

func (m *MemberGrowthProfileModel) Create(profile *MemberGrowthProfile) error {
	return m.CreateWithTx(nil, profile)
}

func (m *MemberGrowthProfileModel) CreateWithTx(tx *gorm.DB, profile *MemberGrowthProfile) error {
	db := m.db
	if tx != nil {
		db = tx
	}
	if db == nil {
		return ErrMemberGrowthProfileDBNil
	}
	if profile == nil {
		return ErrMemberGrowthProfileNil
	}
	return db.Create(profile).Error
}

func (m *MemberGrowthProfileModel) FindByUserId(userId int64) (*MemberGrowthProfile, error) {
	return m.FindByUserIdWithTx(nil, userId)
}

func (m *MemberGrowthProfileModel) FindByUserIdWithTx(tx *gorm.DB, userId int64) (*MemberGrowthProfile, error) {
	if m == nil || m.db == nil {
		return nil, ErrMemberGrowthProfileDBNil
	}
	db := m.db
	if tx != nil {
		db = tx
	}
	var profile MemberGrowthProfile
	err := db.Where("user_id = ?", userId).First(&profile).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (m *MemberGrowthProfileModel) FindOrCreate(userId int64) (*MemberGrowthProfile, error) {
	return m.FindOrCreateWithTx(nil, userId)
}

func (m *MemberGrowthProfileModel) FindOrCreateWithTx(tx *gorm.DB, userId int64) (*MemberGrowthProfile, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	if db == nil {
		return nil, ErrMemberGrowthProfileDBNil
	}
	profile := &MemberGrowthProfile{
		UserId:       userId,
		GrowthLevel:  1,
		GrowthPoints: 0,
	}
	if err := db.Where("user_id = ?", userId).FirstOrCreate(profile).Error; err != nil {
		return nil, err
	}
	return profile, nil
}

func (m *MemberGrowthProfileModel) Update(profile *MemberGrowthProfile) error {
	return m.UpdateWithTx(nil, profile)
}

func (m *MemberGrowthProfileModel) UpdateWithTx(tx *gorm.DB, profile *MemberGrowthProfile) error {
	db := m.db
	if tx != nil {
		db = tx
	}
	if db == nil {
		return ErrMemberGrowthProfileDBNil
	}
	if profile == nil {
		return ErrMemberGrowthProfileNil
	}
	return db.Save(profile).Error
}

type MemberGrowthLogModel struct {
	db *gorm.DB
}

func NewMemberGrowthLogModel(db *gorm.DB) *MemberGrowthLogModel {
	return &MemberGrowthLogModel{db: db}
}

func (m *MemberGrowthLogModel) Create(log *MemberGrowthLog) error {
	return m.CreateWithTx(nil, log)
}

func (m *MemberGrowthLogModel) CreateWithTx(tx *gorm.DB, log *MemberGrowthLog) error {
	db := m.db
	if tx != nil {
		db = tx
	}
	if db == nil {
		return ErrMemberGrowthLogDBNil
	}
	if log == nil {
		return ErrMemberGrowthLogNil
	}
	log.Source = strings.TrimSpace(log.Source)
	if log.Source == "" {
		return errors.New("member growth source is required")
	}
	return db.Create(log).Error
}

func (m *MemberGrowthLogModel) FindByUserMatchAndSource(userId, matchId int64, source string) (*MemberGrowthLog, error) {
	if m == nil || m.db == nil {
		return nil, ErrMemberGrowthLogDBNil
	}
	var log MemberGrowthLog
	err := m.db.Where("user_id = ? AND match_id = ? AND source = ?", userId, matchId, strings.TrimSpace(source)).First(&log).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &log, nil
}
