package model

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	ReputationChangeTypePenalty      = "penalty"
	ReputationChangeTypeRecovery     = "recovery"
	ReputationChangeTypeManualAdjust = "manual_adjust"

	ReputationReasonDurationAbnormal          = "duration_abnormal"
	ReputationReasonSameOpponentHighFrequency = "same_opponent_high_frequency"
	ReputationReasonSystemRecovery            = "system_recovery"
)

var (
	ErrUserReputationLogDBNil = errors.New("user reputation log db is nil")
	ErrUserReputationLogNil   = errors.New("user reputation log is nil")
)

type UserReputationLog struct {
	ID              int64     `gorm:"primarykey" json:"id"`
	UserID          int64     `gorm:"column:user_id;not null;uniqueIndex:uniq_user_match_change_type_reason_code,priority:1;index" json:"user_id"`
	MatchID         *int64    `gorm:"column:match_id;uniqueIndex:uniq_user_match_change_type_reason_code,priority:2" json:"match_id"`
	ChangeType      string    `gorm:"column:change_type;size:32;not null;uniqueIndex:uniq_user_match_change_type_reason_code,priority:3" json:"change_type"`
	ChangeScore     int       `gorm:"column:change_score;not null" json:"change_score"`
	BeforeScore     int       `gorm:"column:before_score;not null" json:"before_score"`
	AfterScore      int       `gorm:"column:after_score;not null" json:"after_score"`
	ReasonCode      string    `gorm:"column:reason_code;size:64;not null;uniqueIndex:uniq_user_match_change_type_reason_code,priority:4" json:"reason_code"`
	ReasonDetail    string    `gorm:"column:reason_detail;type:text" json:"reason_detail"`
	OperatorAdminID int64     `gorm:"column:operator_admin_id;not null;default:0" json:"operator_admin_id"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (UserReputationLog) TableName() string {
	return "user_reputation_logs"
}

type UserReputationLogModel struct {
	db *gorm.DB
}

type UserReputationLogAdminFilter struct {
	Page       int
	PageSize   int
	UserID     int64
	ChangeType string
	ReasonCode string
}

func NewUserReputationLogModel(db *gorm.DB) *UserReputationLogModel {
	return &UserReputationLogModel{db: db}
}

func (m *UserReputationLogModel) Create(log *UserReputationLog) error {
	return m.CreateWithTx(nil, log)
}

func (m *UserReputationLogModel) CreateWithTx(tx *gorm.DB, log *UserReputationLog) error {
	if m == nil {
		return ErrUserReputationLogDBNil
	}
	db := m.db
	if tx != nil {
		db = tx
	}
	if db == nil {
		return ErrUserReputationLogDBNil
	}
	if log == nil {
		return ErrUserReputationLogNil
	}

	log.ChangeType = strings.TrimSpace(log.ChangeType)
	log.ReasonCode = strings.TrimSpace(log.ReasonCode)
	if log.ChangeType == "" {
		return errors.New("reputation change type is required")
	}
	if log.ReasonCode == "" {
		return errors.New("reputation reason code is required")
	}
	return db.Create(log).Error
}

func (m *UserReputationLogModel) FindByUserMatchChangeTypeAndReasonCode(userID, matchID int64, changeType, reasonCode string) (*UserReputationLog, error) {
	if m == nil || m.db == nil {
		return nil, ErrUserReputationLogDBNil
	}

	changeType = strings.TrimSpace(changeType)
	reasonCode = strings.TrimSpace(reasonCode)
	query := m.db.Where("user_id = ? AND match_id = ? AND change_type = ? AND reason_code = ?", userID, matchID, changeType, reasonCode)

	var log UserReputationLog
	err := query.First(&log).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (m *UserReputationLogModel) FindListForAdmin(filter UserReputationLogAdminFilter) ([]UserReputationLog, int64, error) {
	if m == nil || m.db == nil {
		return nil, 0, ErrUserReputationLogDBNil
	}

	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize
	query := m.db.Model(&UserReputationLog{})

	if filter.UserID > 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if changeType := strings.TrimSpace(filter.ChangeType); changeType != "" {
		query = query.Where("change_type = ?", changeType)
	}
	if reasonCode := strings.TrimSpace(filter.ReasonCode); reasonCode != "" {
		query = query.Where("reason_code = ?", reasonCode)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []UserReputationLog
	err := query.Order("created_at DESC, id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (m *UserReputationLogModel) FindListForUser(userID int64, page, pageSize int) ([]UserReputationLog, int64, error) {
	if m == nil || m.db == nil {
		return nil, 0, ErrUserReputationLogDBNil
	}

	page, pageSize = normalizePage(page, pageSize)
	offset := (page - 1) * pageSize
	query := m.db.Model(&UserReputationLog{}).Where("user_id = ?", userID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []UserReputationLog
	err := query.Order("created_at DESC, id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func ReputationChangeTypeText(changeType string) string {
	switch strings.TrimSpace(changeType) {
	case ReputationChangeTypePenalty:
		return "扣分"
	case ReputationChangeTypeRecovery:
		return "恢复"
	case ReputationChangeTypeManualAdjust:
		return "人工调整"
	default:
		return "信誉变更"
	}
}

func ReputationReasonText(reasonCode, changeType string) string {
	switch strings.TrimSpace(reasonCode) {
	case ReputationReasonDurationAbnormal:
		return "时长异常"
	case ReputationReasonSameOpponentHighFrequency:
		return "同对手高频"
	case ReputationReasonSystemRecovery:
		return "系统恢复"
	}

	switch strings.TrimSpace(changeType) {
	case ReputationChangeTypeRecovery:
		return "系统恢复"
	case ReputationChangeTypeManualAdjust:
		return "人工调整"
	default:
		return "信誉变更"
	}
}

func UserFacingReputationReasonCode(reasonCode string) string {
	switch strings.TrimSpace(reasonCode) {
	case ReputationReasonDurationAbnormal:
		return ReputationReasonDurationAbnormal
	case ReputationReasonSameOpponentHighFrequency:
		return ReputationReasonSameOpponentHighFrequency
	case ReputationReasonSystemRecovery:
		return ReputationReasonSystemRecovery
	default:
		return "other"
	}
}
