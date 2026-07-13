package testsupport

import (
	"time"

	"gorm.io/gorm"
)

type socialUserSchema struct {
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

func (socialUserSchema) TableName() string {
	return "users"
}

type socialFollowSchema struct {
	Id          int64     `gorm:"primarykey"`
	FollowerId  int64     `gorm:"not null;index:idx_follow,unique"`
	FollowingId int64     `gorm:"not null;index:idx_follow,unique;index:idx_following"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

func (socialFollowSchema) TableName() string {
	return "follows"
}

type socialPostSchema struct {
	Id            int64   `gorm:"primarykey"`
	UserId        int64   `gorm:"not null;index"`
	Content       string  `gorm:"size:2000;not null"`
	Images        *string `gorm:"type:json"`
	PostType      int     `gorm:"not null;default:3"`
	MatchId       *int64  `gorm:"index"`
	LikesCount    int     `gorm:"not null;default:0"`
	CommentsCount int     `gorm:"not null;default:0"`
	Status        int     `gorm:"not null;default:2;index"`
	RejectReason  string  `gorm:"size:255;not null;default:''"`
	ReviewedAt    *time.Time
	ReviewedBy    *int64
	CreatedAt     time.Time `gorm:"autoCreateTime"`
}

func (socialPostSchema) TableName() string {
	return "social_posts"
}

type socialPostLikeSchema struct {
	Id        int64     `gorm:"primarykey"`
	PostId    int64     `gorm:"not null;uniqueIndex:idx_post_user"`
	UserId    int64     `gorm:"not null;uniqueIndex:idx_post_user"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (socialPostLikeSchema) TableName() string {
	return "social_post_likes"
}

type socialPostCommentSchema struct {
	Id        int64     `gorm:"primarykey"`
	PostId    int64     `gorm:"not null;index"`
	UserId    int64     `gorm:"not null"`
	Content   string    `gorm:"size:500;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (socialPostCommentSchema) TableName() string {
	return "social_post_comments"
}

// PrepareSocialSchema bootstraps sqlite-backed tests for social posting flows.
func PrepareSocialSchema(db *gorm.DB) error {
	return db.AutoMigrate(
		&socialUserSchema{},
		&socialFollowSchema{},
		&socialPostSchema{},
		&socialPostLikeSchema{},
		&socialPostCommentSchema{},
	)
}
