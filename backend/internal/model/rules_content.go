package model

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

type RulesContent struct {
	Id          int64     `gorm:"primarykey"`
	Category    string    `gorm:"size:20;not null;index"`
	ContentType string    `gorm:"size:20;not null;index"`
	Title       string    `gorm:"size:200;not null"`
	Content     string    `gorm:"type:text;not null"`
	SortOrder   int       `gorm:"not null;default:0"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

func (RulesContent) TableName() string {
	return "rules_content"
}

type RulesContentModel struct {
	db *gorm.DB
}

func NewRulesContentModel(db *gorm.DB) *RulesContentModel {
	return &RulesContentModel{db: db}
}

func (m *RulesContentModel) FindByCategory(category string) ([]RulesContent, error) {
	var list []RulesContent
	err := m.db.Where("category = ?", category).Order("sort_order ASC, id ASC").Find(&list).Error
	return list, err
}

func (m *RulesContentModel) FindByCategoryAndType(category, contentType string) ([]RulesContent, error) {
	var list []RulesContent
	err := m.db.Where("category = ? AND content_type = ?", category, contentType).Order("sort_order ASC, id ASC").Find(&list).Error
	return list, err
}

func (m *RulesContentModel) FindGlossary(category string) ([]RulesContent, error) {
	var list []RulesContent
	query := m.db.Where("content_type = ?", "glossary")
	if strings.TrimSpace(category) != "" {
		query = query.Where("category = ?", category)
	}
	err := query.Order("sort_order ASC, id ASC").Find(&list).Error
	return list, err
}

func (m *RulesContentModel) Search(keyword string) ([]RulesContent, error) {
	trimmed := strings.TrimSpace(keyword)
	if trimmed == "" {
		return []RulesContent{}, nil
	}

	var list []RulesContent
	kw := "%" + trimmed + "%"
	err := m.db.Where("title LIKE ? OR content LIKE ?", kw, kw).Order("sort_order ASC, id ASC").Find(&list).Error
	return list, err
}

func (m *RulesContentModel) SeedData(data []RulesContent) error {
	if len(data) == 0 {
		return nil
	}

	query := m.db.Model(&RulesContent{}).Select("category", "content_type", "title")
	for i, item := range data {
		if i == 0 {
			query = query.Where("(category = ? AND content_type = ? AND title = ?)", item.Category, item.ContentType, item.Title)
			continue
		}
		query = query.Or("(category = ? AND content_type = ? AND title = ?)", item.Category, item.ContentType, item.Title)
	}

	var existing []RulesContent
	if err := query.Find(&existing).Error; err != nil {
		return err
	}

	exists := make(map[string]struct{}, len(existing))
	for _, item := range existing {
		key := item.Category + "|" + item.ContentType + "|" + item.Title
		exists[key] = struct{}{}
	}

	insertList := make([]RulesContent, 0, len(data))
	for _, item := range data {
		key := item.Category + "|" + item.ContentType + "|" + item.Title
		if _, ok := exists[key]; ok {
			continue
		}
		insertList = append(insertList, item)
	}

	if len(insertList) == 0 {
		return nil
	}

	return m.db.Create(&insertList).Error
}
