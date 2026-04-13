package model

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

const DefaultMemberRightsConfigKey = "member_rights"

var (
	ErrMemberRightsConfigDBNil = errors.New("member rights config db is nil")
	ErrMemberRightsConfigNil   = errors.New("member rights config is nil")
)

type MemberGrowthRulesConfig struct {
	PointsPerCompletedMatch int    `json:"points_per_completed_match"`
	DailyCap               int    `json:"daily_cap"`
	LevelThresholdLv2      int    `json:"level_threshold_lv2"`
	LevelThresholdLv3      int    `json:"level_threshold_lv3"`
	LevelThresholdLv4      int    `json:"level_threshold_lv4"`
	LevelThresholdLv5      int    `json:"level_threshold_lv5"`
	ExpireStrategy         string `json:"expire_strategy"`
}

type MemberRankingRightsRulesConfig struct {
	OrdinaryUserAchievementEnabled bool `json:"ordinary_user_achievement_enabled"`
	DailyCap                      int  `json:"daily_cap"`
	DailyPositiveCap              int  `json:"daily_positive_cap"`
	Break50Score                  int  `json:"break_50_score"`
	GoldenBreakScore              int  `json:"golden_break_score"`
	BreakAndRunScore              int  `json:"break_and_run_score"`
	RunOutScore                   int  `json:"run_out_score"`
	Break100Score                 int  `json:"break_100_score"`
	NineOnBreakScore              int  `json:"nine_on_break_score"`
	Break147Score                 int  `json:"break_147_score"`
	Level1Multiplier              int  `json:"level1_multiplier"`
	Level2Multiplier              int  `json:"level2_multiplier"`
	Level3Multiplier              int  `json:"level3_multiplier"`
	Level4Multiplier              int  `json:"level4_multiplier"`
	Level5Multiplier              int  `json:"level5_multiplier"`
}

type MemberRightsConfig struct {
	Id                     int64     `gorm:"primarykey" json:"id"`
	ConfigKey              string    `gorm:"size:64;not null;uniqueIndex" json:"config_key"`
	GrowthRulesJSON        string    `gorm:"type:json;not null" json:"growth_rules_json"`
	RankingRightsRulesJSON string    `gorm:"type:json;not null" json:"ranking_rights_rules_json"`
	UpdatedBy              int64     `gorm:"not null;default:0" json:"updated_by"`
	CreatedAt              time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt              time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (MemberRightsConfig) TableName() string {
	return "member_rights_configs"
}

func DefaultMemberGrowthRulesConfig() MemberGrowthRulesConfig {
	return MemberGrowthRulesConfig{
		PointsPerCompletedMatch: 1,
		DailyCap:               5,
		LevelThresholdLv2:      10,
		LevelThresholdLv3:      60,
		LevelThresholdLv4:      260,
		LevelThresholdLv5:      760,
		ExpireStrategy:         "freeze_preserve",
	}
}

func DefaultMemberRankingRightsRulesConfig() MemberRankingRightsRulesConfig {
	return MemberRankingRightsRulesConfig{
		OrdinaryUserAchievementEnabled: false,
		DailyCap:                      200,
		DailyPositiveCap:              500,
		Break50Score:                  8,
		GoldenBreakScore:              4,
		BreakAndRunScore:              6,
		RunOutScore:                   4,
		Break100Score:                 16,
		NineOnBreakScore:              6,
		Break147Score:                 30,
		Level1Multiplier:              100,
		Level2Multiplier:              110,
		Level3Multiplier:              120,
		Level4Multiplier:              130,
		Level5Multiplier:              140,
	}
}

func DefaultMemberRightsConfig() *MemberRightsConfig {
	cfg := &MemberRightsConfig{
		ConfigKey: DefaultMemberRightsConfigKey,
		UpdatedBy: 0,
	}
	cfg.SetGrowthRules(DefaultMemberGrowthRulesConfig())
	cfg.SetRankingRightsRules(DefaultMemberRankingRightsRulesConfig())
	return cfg
}

func (c *MemberRightsConfig) SetGrowthRules(rules MemberGrowthRulesConfig) {
	data, _ := json.Marshal(rules)
	c.GrowthRulesJSON = string(data)
}

func (c *MemberRightsConfig) SetRankingRightsRules(rules MemberRankingRightsRulesConfig) {
	data, _ := json.Marshal(rules)
	c.RankingRightsRulesJSON = string(data)
}

func (c *MemberRightsConfig) GrowthRules() (MemberGrowthRulesConfig, error) {
	rules := DefaultMemberGrowthRulesConfig()
	if c == nil || strings.TrimSpace(c.GrowthRulesJSON) == "" {
		return rules, nil
	}
	err := json.Unmarshal([]byte(c.GrowthRulesJSON), &rules)
	return rules, err
}

func (c *MemberRightsConfig) RankingRightsRules() (MemberRankingRightsRulesConfig, error) {
	rules := DefaultMemberRankingRightsRulesConfig()
	if c == nil || strings.TrimSpace(c.RankingRightsRulesJSON) == "" {
		return rules, nil
	}
	err := json.Unmarshal([]byte(c.RankingRightsRulesJSON), &rules)
	return rules, err
}

type MemberRightsConfigModel struct {
	db *gorm.DB
}

func NewMemberRightsConfigModel(db *gorm.DB) *MemberRightsConfigModel {
	return &MemberRightsConfigModel{db: db}
}

func (m *MemberRightsConfigModel) FindByKey(configKey string) (*MemberRightsConfig, error) {
	if m == nil || m.db == nil {
		return nil, ErrMemberRightsConfigDBNil
	}
	var cfg MemberRightsConfig
	err := m.db.Where("config_key = ?", strings.TrimSpace(configKey)).First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (m *MemberRightsConfigModel) Upsert(cfg *MemberRightsConfig) error {
	if m == nil || m.db == nil {
		return ErrMemberRightsConfigDBNil
	}
	if cfg == nil {
		return ErrMemberRightsConfigNil
	}
	cfg.ConfigKey = strings.TrimSpace(cfg.ConfigKey)
	if cfg.ConfigKey == "" {
		cfg.ConfigKey = DefaultMemberRightsConfigKey
	}

	existing, err := m.FindByKey(cfg.ConfigKey)
	if err != nil {
		return err
	}
	if existing == nil {
		return m.db.Create(cfg).Error
	}

	return m.db.Model(&MemberRightsConfig{}).
		Where("id = ?", existing.Id).
		Updates(map[string]interface{}{
			"growth_rules_json":         cfg.GrowthRulesJSON,
			"ranking_rights_rules_json": cfg.RankingRightsRulesJSON,
			"updated_by":                cfg.UpdatedBy,
		}).Error
}
