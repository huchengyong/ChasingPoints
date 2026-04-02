package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	EventNewsStatusUpcoming = 0
	EventNewsStatusLive     = 1
	EventNewsStatusFinished = 2
	EventNewsStatusCanceled = 3
)

type EventNews struct {
	Id           int64          `gorm:"primarykey" json:"id"`
	Title        string         `gorm:"size:128;not null" json:"title"`
	TournamentId int64          `gorm:"not null;default:0;index" json:"tournament_id"`
	GameType     int            `gorm:"not null;index" json:"game_type"`
	SourceType   string         `gorm:"size:32;not null;default:''" json:"source_type"`
	SourceName   string         `gorm:"size:64;not null;default:''" json:"source_name"`
	SourceUrl    string         `gorm:"size:512;not null;default:''" json:"source_url"`
	CoverImage   string         `gorm:"size:512;not null;default:''" json:"cover_image"`
	Summary      string         `gorm:"size:512;not null;default:''" json:"summary"`
	Content      string         `gorm:"type:text" json:"content"`
	Country      string         `gorm:"size:64;not null;default:''" json:"country"`
	City         string         `gorm:"size:64;not null;default:'';index" json:"city"`
	Venue        string         `gorm:"size:128;not null;default:''" json:"venue"`
	StartTime    *time.Time     `gorm:"default:null;index" json:"start_time"`
	EndTime      *time.Time     `gorm:"default:null" json:"end_time"`
	Status       int            `gorm:"not null;default:0;index" json:"status"`
	Featured     bool           `gorm:"not null;default:false;index" json:"featured"`
	SortTime     *time.Time     `gorm:"default:null;index" json:"sort_time"`
	Published    bool           `gorm:"not null;default:false;index" json:"published"`
	PublishedAt  *time.Time     `gorm:"default:null" json:"published_at"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

func (EventNews) TableName() string {
	return "event_news_events"
}

type EventNewsModel struct {
	db *gorm.DB
}

func NewEventNewsModel(db *gorm.DB) *EventNewsModel {
	return &EventNewsModel{db: db}
}

func (m *EventNewsModel) Create(eventNews *EventNews) error {
	return m.db.Create(eventNews).Error
}

func (m *EventNewsModel) Update(eventNews *EventNews) error {
	return m.db.Save(eventNews).Error
}

func (m *EventNewsModel) UpdateTournamentBinding(id, tournamentId int64) error {
	return m.db.Model(&EventNews{}).
		Where("id = ?", id).
		Update("tournament_id", tournamentId).
		Error
}

func (m *EventNewsModel) FindById(id int64) (*EventNews, error) {
	var eventNews EventNews
	err := m.db.First(&eventNews, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &eventNews, err
}

func (m *EventNewsModel) FindPublishedById(id int64) (*EventNews, error) {
	var eventNews EventNews
	err := m.db.Where("id = ? AND published = ?", id, true).First(&eventNews).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &eventNews, err
}

func (m *EventNewsModel) FindPublishedMatching(gameType, status int, city string) ([]EventNews, error) {
	query := m.db.Model(&EventNews{}).Where("published = ?", true)
	if gameType > 0 {
		query = query.Where("game_type = ?", gameType)
	}
	if status >= 0 {
		query = query.Where("status = ?", status)
	}
	if city != "" {
		query = query.Where("city = ?", city)
	}

	var list []EventNews
	err := query.Order("featured DESC, sort_time ASC, id DESC").Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (m *EventNewsModel) FindList(page, pageSize int, gameType, status int, city string, published int) ([]EventNews, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	offset := (page - 1) * pageSize

	query := m.db.Model(&EventNews{})
	if gameType > 0 {
		query = query.Where("game_type = ?", gameType)
	}
	if status >= 0 {
		query = query.Where("status = ?", status)
	}
	if city != "" {
		query = query.Where("city = ?", city)
	}
	if published >= 0 {
		query = query.Where("published = ?", published == 1)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []EventNews
	err := query.Order("featured DESC, sort_time ASC, id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (m *EventNewsModel) FindFeatured() (*EventNews, error) {
	var eventNews EventNews
	err := m.db.Where("featured = 1 AND published = 1").Order("sort_time ASC, id DESC").First(&eventNews).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &eventNews, nil
}

func (m *EventNewsModel) UpdatePublished(id int64, published bool) (bool, error) {
	updates := map[string]interface{}{
		"published": published,
	}
	if published {
		now := time.Now()
		updates["published_at"] = &now
	} else {
		updates["published_at"] = nil
	}

	result := m.db.Model(&EventNews{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (m *EventNewsModel) SoftDelete(id int64) (bool, error) {
	result := m.db.Where("id = ?", id).Delete(&EventNews{})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}
