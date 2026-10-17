package model

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const DefaultLlmConfigKey = "default"

// 大模型接口格式：anthropic 为 Anthropic Messages；openai 为 OpenAI 兼容的 Chat Completions。
const (
	LlmAPIFormatAnthropic = "anthropic"
	LlmAPIFormatOpenAI    = "openai"
)

var (
	ErrLlmConfigDBNil = errors.New("llm config db is nil")
	ErrLlmConfigNil   = errors.New("llm config is nil")
)

// LlmConfig 保存大模型调用所需的接口地址、鉴权与默认参数；ApiKey 为敏感字段，不得直接返回给客户端或写入日志。
type LlmConfig struct {
	ID          int64     `gorm:"primarykey" json:"id"`
	ConfigKey   string    `gorm:"size:64;not null;uniqueIndex" json:"config_key"`
	APIFormat   string    `gorm:"column:api_format;size:32;not null" json:"api_format"`
	EndpointURL string    `gorm:"size:512;not null" json:"endpoint_url"`
	APIKey      string    `gorm:"column:api_key;size:512;not null" json:"-"`
	Model       string    `gorm:"size:128;not null" json:"model"`
	MaxTokens   int       `gorm:"not null" json:"max_tokens"`
	UpdatedBy   int64     `gorm:"not null;default:0" json:"updated_by"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (LlmConfig) TableName() string {
	return "llm_configs"
}

type LlmConfigModel struct {
	db *gorm.DB
}

func NewLlmConfigModel(db *gorm.DB) *LlmConfigModel {
	return &LlmConfigModel{db: db}
}

// FindByKey 未配置时返回 (nil, nil)，由调用方决定如何呈现“尚未配置”。
func (m *LlmConfigModel) FindByKey(configKey string) (*LlmConfig, error) {
	if m == nil || m.db == nil {
		return nil, ErrLlmConfigDBNil
	}

	var cfg LlmConfig
	err := m.db.Where("config_key = ?", strings.TrimSpace(configKey)).First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (m *LlmConfigModel) Upsert(cfg *LlmConfig) error {
	if m == nil || m.db == nil {
		return ErrLlmConfigDBNil
	}
	if cfg == nil {
		return ErrLlmConfigNil
	}

	cfg.ConfigKey = strings.TrimSpace(cfg.ConfigKey)
	if cfg.ConfigKey == "" {
		cfg.ConfigKey = DefaultLlmConfigKey
	}

	return m.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "config_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"api_format", "endpoint_url", "api_key", "model", "max_tokens", "updated_by", "updated_at"}),
	}).Create(cfg).Error
}
