package testsupport

import (
	"time"

	"gorm.io/gorm"
)

type eventNewsSchema struct {
	Id          int64          `gorm:"primarykey"`
	Title       string         `gorm:"size:128;not null"`
	GameType    int            `gorm:"not null;index"`
	SourceType  string         `gorm:"size:32;not null;default:''"`
	SourceName  string         `gorm:"size:64;not null;default:''"`
	SourceUrl   string         `gorm:"size:512;not null;default:''"`
	CoverImage  string         `gorm:"size:512;not null;default:''"`
	Summary     string         `gorm:"size:512;not null;default:''"`
	Content     string         `gorm:"type:text"`
	Country     string         `gorm:"size:64;not null;default:''"`
	City        string         `gorm:"size:64;not null;default:'';index"`
	Venue       string         `gorm:"size:128;not null;default:''"`
	StartTime   *time.Time     `gorm:"default:null;index"`
	EndTime     *time.Time     `gorm:"default:null"`
	Status      int            `gorm:"not null;default:0;index"`
	StageText   string         `gorm:"size:128;not null;default:''"`
	ResultText  string         `gorm:"size:255;not null;default:''"`
	Featured    bool           `gorm:"not null;default:false;index"`
	SortTime    *time.Time     `gorm:"default:null;index"`
	Published   bool           `gorm:"not null;default:false;index"`
	PublishedAt *time.Time     `gorm:"default:null"`
	CreatedAt   time.Time      `gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (eventNewsSchema) TableName() string {
	return "event_news"
}

// PrepareEventNewsSchema bootstraps the event_news table for sqlite-backed tests.
func PrepareEventNewsSchema(db *gorm.DB) error {
	return db.AutoMigrate(&eventNewsSchema{})
}
