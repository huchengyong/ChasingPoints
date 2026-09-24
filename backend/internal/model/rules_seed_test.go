package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDefaultSnookerRulesMatchOfficialCorrections(t *testing.T) {
	expected := map[string]string{
		"rule|开球规则":      "不要求该杆必须有红球碰库或入袋",
		"rule|斯诺克规则":     "目标球的两个极边",
		"foul|同时击中两球":    "同时首碰两颗红球合法",
		"glossary|斯诺克":   "目标球两个极边",
		"glossary|满分147": "155分",
		"glossary|自由球":   "指定一颗非目标球",
	}
	for _, item := range GetDefaultRulesContent() {
		if item.Category != "snooker" {
			continue
		}
		key := item.ContentType + "|" + item.Title
		want, ok := expected[key]
		if !ok {
			continue
		}
		if !strings.Contains(item.Content, want) {
			t.Fatalf("%s missing %q: %s", key, want, item.Content)
		}
		delete(expected, key)
	}
	if len(expected) != 0 {
		t.Fatalf("missing corrected rules: %+v", expected)
	}
}

func TestRulesSeedCreatesCorrectSnookerContent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&RulesContent{}); err != nil {
		t.Fatalf("prepare schema: %v", err)
	}
	model := NewRulesContentModel(db)
	if err := model.SeedData(GetDefaultRulesContent()); err != nil {
		t.Fatalf("seed rules: %v", err)
	}
	rows, err := model.FindByCategoryAndType("snooker", "glossary")
	if err != nil {
		t.Fatalf("load glossary: %v", err)
	}
	found155 := false
	for _, row := range rows {
		if row.Title == "满分147" && strings.Contains(row.Content, "155分") {
			found155 = true
		}
	}
	if !found155 {
		t.Fatal("new database did not receive corrected 147/155 content")
	}
}

func TestSnookerRulesMigrationUpdatesExistingRowsAndHasDown(t *testing.T) {
	path := filepath.Join("..", "..", "migrations", "20260730121000_update_snooker_rules_content.sql")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	content := string(data)
	for _, required := range []string{
		"-- +goose Up",
		"-- +goose Down",
		"AND `title` = '开球规则'",
		"AND `title` = '斯诺克规则'",
		"AND `title` = '同时击中两球'",
		"AND `title` = '满分147'",
		"AND `title` = '自由球'",
		"理论最高单杆可达155分",
		"至少有一颗红球碰库或入袋",
	} {
		if !strings.Contains(content, required) {
			t.Fatalf("migration missing %q", required)
		}
	}
}
