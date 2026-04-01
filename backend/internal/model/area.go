package model

import (
	"errors"

	"gorm.io/gorm"
)

type Area struct {
	AreaId   int64  `gorm:"primarykey;column:area_id" json:"area_id"`
	ParentId int64  `gorm:"column:parent_id;not null;default:0;index" json:"parent_id"`
	Name     string `gorm:"column:name;size:120;not null;default:''" json:"name"`
}

func (Area) TableName() string {
	return "dou_area"
}

type AreaModel struct {
	db *gorm.DB
}

func NewAreaModel(db *gorm.DB) *AreaModel {
	return &AreaModel{db: db}
}

func (m *AreaModel) FindChildren(parentId int64) ([]Area, error) {
	if m.db == nil {
		return nil, errors.New("area db is nil")
	}
	if parentId < 0 {
		parentId = 0
	}

	var list []Area
	err := m.db.Where("parent_id = ?", parentId).Order("area_id ASC").Find(&list).Error
	return list, err
}
