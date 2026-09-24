package testsupport

import (
	"time"

	"gorm.io/gorm"
)

type friendUserSchema struct {
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

func (friendUserSchema) TableName() string {
	return "users"
}

type friendSchema struct {
	Id        int64     `gorm:"primarykey"`
	UserId    int64     `gorm:"not null;index:idx_user_friend,unique"`
	FriendId  int64     `gorm:"not null;index:idx_user_friend,unique"`
	Status    int       `gorm:"not null;default:1"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (friendSchema) TableName() string {
	return "friends"
}

type friendRequestSchema struct {
	Id         int64     `gorm:"primarykey"`
	FromUserId int64     `gorm:"not null;index"`
	ToUserId   int64     `gorm:"not null;index"`
	Status     int       `gorm:"not null;default:0"`
	Message    string    `gorm:"size:200"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

func (friendRequestSchema) TableName() string {
	return "friend_requests"
}

type friendBlacklistSchema struct {
	Id            int64     `gorm:"primarykey"`
	UserId        int64     `gorm:"not null;index:idx_user_blocked,unique"`
	BlockedUserId int64     `gorm:"not null;index:idx_user_blocked,unique;index:idx_blocked_user"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
}

func (friendBlacklistSchema) TableName() string {
	return "friend_blacklists"
}

type friendNotificationSchema struct {
	Id        int64     `gorm:"primarykey"`
	UserId    int64     `gorm:"not null;index;uniqueIndex:uk_notifications_user_type_dedupe,priority:1"`
	Type      string    `gorm:"size:30;not null;uniqueIndex:uk_notifications_user_type_dedupe,priority:2"`
	DedupeKey *string   `gorm:"size:128;uniqueIndex:uk_notifications_user_type_dedupe,priority:3"`
	Title     string    `gorm:"size:200;not null"`
	Content   string    `gorm:"size:500"`
	Data      *string   `gorm:"type:json"`
	IsRead    int       `gorm:"not null;default:0"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (friendNotificationSchema) TableName() string {
	return "notifications"
}

func PrepareFriendSchema(db *gorm.DB) error {
	return db.AutoMigrate(
		&friendUserSchema{},
		&friendSchema{},
		&friendRequestSchema{},
		&friendBlacklistSchema{},
		&friendNotificationSchema{},
	)
}
