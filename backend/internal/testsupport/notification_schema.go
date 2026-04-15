package testsupport

import (
	"time"

	"gorm.io/gorm"
)

type notificationUserSchema struct {
	Id              int64   `gorm:"primarykey"`
	Phone           *string `gorm:"uniqueIndex;size:20"`
	Nickname        string  `gorm:"size:50;not null;default:''"`
	Avatar          string  `gorm:"size:255;not null;default:''"`
	Status          int     `gorm:"not null;default:1"`
	PushToken       string  `gorm:"size:255;not null;default:''"`
	MemberExpiresAt *time.Time
	HideMatchRecord bool           `gorm:"not null;default:false"`
	CreatedAt       time.Time      `gorm:"autoCreateTime"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

func (notificationUserSchema) TableName() string {
	return "users"
}

type notificationPreferenceSchema struct {
	Id                   int64     `gorm:"primarykey"`
	UserId               int64     `gorm:"not null;uniqueIndex"`
	MatchResultEnabled   bool      `gorm:"not null;default:true"`
	FriendRequestEnabled bool      `gorm:"not null;default:true"`
	ChallengeEnabled     bool      `gorm:"not null;default:true"`
	TournamentEnabled    bool      `gorm:"not null;default:true"`
	FollowEnabled        bool      `gorm:"not null;default:true"`
	CreatedAt            time.Time `gorm:"autoCreateTime"`
	UpdatedAt            time.Time `gorm:"autoUpdateTime"`
}

func (notificationPreferenceSchema) TableName() string {
	return "user_notification_preferences"
}

type notificationMessageSchema struct {
	Id        int64     `gorm:"primarykey"`
	UserId    int64     `gorm:"not null;index"`
	Type      string    `gorm:"size:30;not null"`
	Title     string    `gorm:"size:200;not null"`
	Content   string    `gorm:"size:500"`
	Data      *string   `gorm:"type:json"`
	IsRead    int       `gorm:"not null;default:0"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (notificationMessageSchema) TableName() string {
	return "notifications"
}

func PrepareNotificationSchema(db *gorm.DB) error {
	return db.AutoMigrate(&notificationUserSchema{}, &notificationPreferenceSchema{}, &notificationMessageSchema{})
}
