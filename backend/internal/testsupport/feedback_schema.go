package testsupport

import (
	"time"

	"gorm.io/gorm"
)

type feedbackTicketSchema struct {
	Id            int64  `gorm:"primarykey"`
	UserId        *int64 `gorm:"index"`
	Source        string `gorm:"size:32;not null;index"`
	Category      string `gorm:"size:32;not null;index"`
	Content       string `gorm:"type:text;not null"`
	Contact       string `gorm:"size:128;not null;default:''"`
	Status        int    `gorm:"not null;default:1;index"`
	HandlerId     *int64
	ProcessResult string `gorm:"type:text;not null;default:''"`
	ProcessedAt   *time.Time
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
	DeletedAt     gorm.DeletedAt
}

func (feedbackTicketSchema) TableName() string {
	return "feedback_tickets"
}

// PrepareFeedbackSchema bootstraps sqlite-backed tests for feedback ticket flows.
func PrepareFeedbackSchema(db *gorm.DB) error {
	return db.AutoMigrate(&feedbackTicketSchema{})
}
