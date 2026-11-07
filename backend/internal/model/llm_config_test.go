package model

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newLlmConfigTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&LlmConfig{}); err != nil {
		t.Fatalf("prepare llm config schema: %v", err)
	}
	return db
}

func TestLlmConfigTableName(t *testing.T) {
	if got := (LlmConfig{}).TableName(); got != "llm_configs" {
		t.Fatalf("expected table name llm_configs, got %s", got)
	}
}

func TestLlmConfigFindByKeyReturnsNilWhenMissing(t *testing.T) {
	cfgModel := NewLlmConfigModel(newLlmConfigTestDB(t))

	cfg, err := cfgModel.FindByKey(DefaultLlmConfigKey)
	if err != nil {
		t.Fatalf("find config: %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil config, got %+v", cfg)
	}
}

func TestLlmConfigUpsertInsertsThenOverwritesSingleRow(t *testing.T) {
	db := newLlmConfigTestDB(t)
	cfgModel := NewLlmConfigModel(db)

	first := &LlmConfig{
		APIFormat:   LlmAPIFormatAnthropic,
		EndpointURL: "https://opencode.ai/inference/anthropic/v1/messages",
		APIKey:      "key-first",
		Model:       "claude-haiku-4-5",
		MaxTokens:   1024,
		UpdatedBy:   1,
	}
	if err := cfgModel.Upsert(first); err != nil {
		t.Fatalf("insert config: %v", err)
	}
	if first.ConfigKey != DefaultLlmConfigKey {
		t.Fatalf("expected default config key to be filled, got %q", first.ConfigKey)
	}

	second := &LlmConfig{
		ConfigKey:   DefaultLlmConfigKey,
		APIFormat:   LlmAPIFormatOpenAI,
		EndpointURL: "https://example.com/v1/chat/completions",
		APIKey:      "key-second",
		Model:       "claude-sonnet-4-5",
		MaxTokens:   2048,
		UpdatedBy:   2,
	}
	if err := cfgModel.Upsert(second); err != nil {
		t.Fatalf("overwrite config: %v", err)
	}

	var count int64
	if err := db.Model(&LlmConfig{}).Count(&count).Error; err != nil {
		t.Fatalf("count configs: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected a single config row, got %d", count)
	}

	stored, err := cfgModel.FindByKey(DefaultLlmConfigKey)
	if err != nil {
		t.Fatalf("find config: %v", err)
	}
	if stored == nil {
		t.Fatal("expected stored config")
	}
	if stored.APIFormat != LlmAPIFormatOpenAI || stored.EndpointURL != second.EndpointURL || stored.APIKey != "key-second" || stored.Model != second.Model || stored.MaxTokens != 2048 || stored.UpdatedBy != 2 {
		t.Fatalf("unexpected stored config: %+v", stored)
	}
}

func TestLlmConfigNeverSerializesAPIKey(t *testing.T) {
	data, err := json.Marshal(&LlmConfig{APIKey: "super-secret-key"})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if strings.Contains(string(data), "super-secret-key") || strings.Contains(string(data), "api_key") {
		t.Fatalf("api key must not be serialized, got %s", data)
	}
}

func TestLlmConfigModelReturnsHelpfulNilErrors(t *testing.T) {
	var cfgModel *LlmConfigModel
	if _, err := cfgModel.FindByKey(DefaultLlmConfigKey); !errors.Is(err, ErrLlmConfigDBNil) {
		t.Fatalf("expected nil db error, got %v", err)
	}
	if err := cfgModel.Upsert(&LlmConfig{}); !errors.Is(err, ErrLlmConfigDBNil) {
		t.Fatalf("expected nil db error, got %v", err)
	}
	if err := NewLlmConfigModel(newLlmConfigTestDB(t)).Upsert(nil); !errors.Is(err, ErrLlmConfigNil) {
		t.Fatalf("expected nil config error, got %v", err)
	}
}
