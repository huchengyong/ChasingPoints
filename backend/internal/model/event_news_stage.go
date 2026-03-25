package model

import (
	"time"

	"gorm.io/gorm"
)

type EventNewsStage struct {
	Id         int64          `gorm:"primarykey" json:"id"`
	EventId    int64          `gorm:"not null;index" json:"event_id"`
	StageName  string         `gorm:"size:128;not null" json:"stage_name"`
	StageOrder int            `gorm:"not null;default:0;index" json:"stage_order"`
	StartTime  *time.Time     `gorm:"default:null;index" json:"start_time"`
	EndTime    *time.Time     `gorm:"default:null" json:"end_time"`
	Status     int            `gorm:"not null;default:0;index" json:"status"`
	ResultText string         `gorm:"size:255;not null;default:''" json:"result_text"`
	SortTime   *time.Time     `gorm:"default:null;index" json:"sort_time"`
	CreatedAt  time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

func (EventNewsStage) TableName() string {
	return "event_news_stages"
}

type EventNewsStageModel struct {
	db *gorm.DB
}

func NewEventNewsStageModel(db *gorm.DB) *EventNewsStageModel {
	return &EventNewsStageModel{db: db}
}

func (m *EventNewsStageModel) Create(stage *EventNewsStage) error {
	return m.db.Create(stage).Error
}

func (m *EventNewsStageModel) Update(stage *EventNewsStage) error {
	return m.db.Save(stage).Error
}

func (m *EventNewsStageModel) FindById(id int64) (*EventNewsStage, error) {
	var stage EventNewsStage
	err := m.db.First(&stage, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &stage, err
}

func (m *EventNewsStageModel) FindByEventId(eventId int64) ([]EventNewsStage, error) {
	var stages []EventNewsStage
	err := m.db.Where("event_id = ?", eventId).Order("stage_order ASC, id ASC").Find(&stages).Error
	if err != nil {
		return nil, err
	}
	if stages == nil {
		stages = []EventNewsStage{}
	}
	return stages, nil
}

func (m *EventNewsStageModel) FindByEventIds(eventIds []int64) (map[int64][]EventNewsStage, error) {
	result := make(map[int64][]EventNewsStage, len(eventIds))
	if len(eventIds) == 0 {
		return result, nil
	}

	var stages []EventNewsStage
	err := m.db.Where("event_id IN ?", eventIds).Order("stage_order ASC, id ASC").Find(&stages).Error
	if err != nil {
		return nil, err
	}

	for _, eventId := range eventIds {
		result[eventId] = []EventNewsStage{}
	}
	for _, stage := range stages {
		result[stage.EventId] = append(result[stage.EventId], stage)
	}
	return result, nil
}

func (m *EventNewsStageModel) SoftDelete(id int64) (bool, error) {
	result := m.db.Where("id = ?", id).Delete(&EventNewsStage{})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (m *EventNewsStageModel) SoftDeleteByEventId(eventId int64) error {
	return m.db.Where("event_id = ?", eventId).Delete(&EventNewsStage{}).Error
}
