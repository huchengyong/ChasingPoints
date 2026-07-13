package model

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

const DefaultReputationConfigKey = "default"

var (
	ErrReputationConfigDBNil = errors.New("reputation config db is nil")
	ErrReputationConfigNil   = errors.New("reputation config is nil")
)

type ReputationBaseRules struct {
	MaxScore         int `json:"max_score"`
	InitialScore     int `json:"initial_score"`
	BanThreshold     int `json:"ban_threshold"`
	BanDurationHours int `json:"ban_duration_hours"`
	MinScore         int `json:"min_score"`
}

type ReputationRecoveryRules struct {
	Enabled         bool `json:"enabled"`
	RecoverPerHour  int  `json:"recover_per_hour"`
	RecoverMaxScore int  `json:"recover_max_score"`
}

type ReputationDurationRule struct {
	GameType                int  `json:"game_type"`
	Enabled                 bool `json:"enabled"`
	MinMinutesPerRound      int  `json:"min_minutes_per_round"`
	MinTotalRounds          int  `json:"min_total_rounds"`
	MinTotalDurationMinutes int  `json:"min_total_duration_minutes"`
	PenaltyScore            int  `json:"penalty_score"`
}

type ReputationSameOpponentRule struct {
	Enabled             bool `json:"enabled"`
	WindowMinutes       int  `json:"window_minutes"`
	MaxMatches          int  `json:"max_matches"`
	PenaltyScore        int  `json:"penalty_score"`
	RequireSameGameType bool `json:"require_same_game_type"`
}

type ReputationDetectionRules struct {
	DurationRules          []ReputationDurationRule   `json:"duration_rules"`
	SameOpponentRule       ReputationSameOpponentRule `json:"same_opponent_rule"`
	StackPenaltiesPerMatch bool                       `json:"stack_penalties_per_match"`
}

type ReputationConfig struct {
	ID                 int64     `gorm:"primarykey" json:"id"`
	ConfigKey          string    `gorm:"size:64;not null;uniqueIndex" json:"config_key"`
	BaseRulesJSON      string    `gorm:"type:json;not null" json:"base_rules_json"`
	RecoveryRulesJSON  string    `gorm:"type:json;not null" json:"recovery_rules_json"`
	DetectionRulesJSON string    `gorm:"type:json;not null" json:"detection_rules_json"`
	UpdatedBy          int64     `gorm:"not null;default:0" json:"updated_by"`
	CreatedAt          time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (ReputationConfig) TableName() string {
	return "reputation_configs"
}

func DefaultReputationBaseRules() ReputationBaseRules {
	return ReputationBaseRules{
		MaxScore:         100,
		InitialScore:     100,
		BanThreshold:     60,
		BanDurationHours: 24,
		MinScore:         0,
	}
}

func DefaultReputationRecoveryRules() ReputationRecoveryRules {
	return ReputationRecoveryRules{
		Enabled:         true,
		RecoverPerHour:  1,
		RecoverMaxScore: 100,
	}
}

func DefaultReputationDetectionRules() ReputationDetectionRules {
	return ReputationDetectionRules{
		DurationRules: []ReputationDurationRule{
			{GameType: 1, Enabled: true, MinMinutesPerRound: 15, MinTotalRounds: 3, MinTotalDurationMinutes: 45, PenaltyScore: 12},
			{GameType: 2, Enabled: true, MinMinutesPerRound: 3, MinTotalRounds: 5, MinTotalDurationMinutes: 18, PenaltyScore: 8},
			{GameType: 3, Enabled: true, MinMinutesPerRound: 2, MinTotalRounds: 5, MinTotalDurationMinutes: 15, PenaltyScore: 10},
			{GameType: 4, Enabled: true, MinMinutesPerRound: 4, MinTotalRounds: 5, MinTotalDurationMinutes: 20, PenaltyScore: 10},
		},
		SameOpponentRule: ReputationSameOpponentRule{
			Enabled:             true,
			WindowMinutes:       30,
			MaxMatches:          4,
			PenaltyScore:        8,
			RequireSameGameType: true,
		},
		StackPenaltiesPerMatch: false,
	}
}

func DefaultReputationConfig() *ReputationConfig {
	cfg := &ReputationConfig{
		ConfigKey: DefaultReputationConfigKey,
		UpdatedBy: 0,
	}
	cfg.SetBaseRules(DefaultReputationBaseRules())
	cfg.SetRecoveryRules(DefaultReputationRecoveryRules())
	cfg.SetDetectionRules(DefaultReputationDetectionRules())
	return cfg
}

func (c *ReputationConfig) SetBaseRules(rules ReputationBaseRules) {
	data, _ := json.Marshal(rules)
	c.BaseRulesJSON = string(data)
}

func (c *ReputationConfig) SetRecoveryRules(rules ReputationRecoveryRules) {
	data, _ := json.Marshal(rules)
	c.RecoveryRulesJSON = string(data)
}

func (c *ReputationConfig) SetDetectionRules(rules ReputationDetectionRules) {
	data, _ := json.Marshal(rules)
	c.DetectionRulesJSON = string(data)
}

func (c *ReputationConfig) BaseRules() (ReputationBaseRules, error) {
	rules := DefaultReputationBaseRules()
	if c == nil || strings.TrimSpace(c.BaseRulesJSON) == "" {
		return rules, nil
	}
	err := json.Unmarshal([]byte(c.BaseRulesJSON), &rules)
	return rules, err
}

func (c *ReputationConfig) RecoveryRules() (ReputationRecoveryRules, error) {
	rules := DefaultReputationRecoveryRules()
	if c == nil || strings.TrimSpace(c.RecoveryRulesJSON) == "" {
		return rules, nil
	}
	err := json.Unmarshal([]byte(c.RecoveryRulesJSON), &rules)
	return rules, err
}

func (c *ReputationConfig) DetectionRules() (ReputationDetectionRules, error) {
	rules := DefaultReputationDetectionRules()
	if c == nil || strings.TrimSpace(c.DetectionRulesJSON) == "" {
		return rules, nil
	}
	err := json.Unmarshal([]byte(c.DetectionRulesJSON), &rules)
	return rules, err
}

type ReputationConfigModel struct {
	db *gorm.DB
}

func NewReputationConfigModel(db *gorm.DB) *ReputationConfigModel {
	return &ReputationConfigModel{db: db}
}

func (m *ReputationConfigModel) FindByKey(configKey string) (*ReputationConfig, error) {
	if m == nil || m.db == nil {
		return nil, ErrReputationConfigDBNil
	}

	var cfg ReputationConfig
	err := m.db.Where("config_key = ?", strings.TrimSpace(configKey)).First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (m *ReputationConfigModel) Upsert(cfg *ReputationConfig) error {
	if m == nil || m.db == nil {
		return ErrReputationConfigDBNil
	}
	if cfg == nil {
		return ErrReputationConfigNil
	}

	cfg.ConfigKey = strings.TrimSpace(cfg.ConfigKey)
	if cfg.ConfigKey == "" {
		cfg.ConfigKey = DefaultReputationConfigKey
	}

	existing, err := m.FindByKey(cfg.ConfigKey)
	if err != nil {
		return err
	}
	if existing == nil {
		return m.db.Create(cfg).Error
	}

	return m.db.Model(&ReputationConfig{}).
		Where("id = ?", existing.ID).
		Updates(map[string]interface{}{
			"base_rules_json":      cfg.BaseRulesJSON,
			"recovery_rules_json":  cfg.RecoveryRulesJSON,
			"detection_rules_json": cfg.DetectionRulesJSON,
			"updated_by":           cfg.UpdatedBy,
		}).Error
}
