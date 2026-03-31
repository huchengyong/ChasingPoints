package testsupport

import (
	"time"

	"gorm.io/gorm"
)

type favoriteVenueRewardUserSchema struct {
	Id              int64          `gorm:"primarykey"`
	Phone           *string        `gorm:"uniqueIndex;size:20"`
	Nickname        string         `gorm:"size:50;not null;default:''"`
	Avatar          string         `gorm:"size:255;not null;default:''"`
	Status          int            `gorm:"not null;default:1"`
	PushToken       string         `gorm:"size:255;not null;default:''"`
	MemberExpiresAt *time.Time     `gorm:"comment:会员到期时间"`
	HideMatchRecord bool           `gorm:"not null;default:false"`
	CreatedAt       time.Time      `gorm:"autoCreateTime"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

func (favoriteVenueRewardUserSchema) TableName() string {
	return "users"
}

type favoriteVenueRewardVenueSchema struct {
	Id                 int64   `gorm:"primarykey"`
	Name               string  `gorm:"size:128;not null"`
	Address            string  `gorm:"size:256;not null"`
	City               string  `gorm:"size:64;not null;index"`
	District           string  `gorm:"size:64;not null;default:''"`
	FullAddress        string  `gorm:"size:512;not null;default:'';uniqueIndex:uniq_full_address"`
	Latitude           float64 `gorm:"not null;default:0"`
	Longitude          float64 `gorm:"not null;default:0"`
	Phone              string  `gorm:"size:32;not null;default:''"`
	Images             string  `gorm:"type:text"`
	BusinessHours      string  `gorm:"size:128;not null;default:''"`
	TableCount         int     `gorm:"not null;default:0"`
	PriceRange         string  `gorm:"size:64;not null;default:''"`
	Description        string  `gorm:"type:text"`
	OwnerUserId        int64   `gorm:"default:null"`
	Status             int     `gorm:"not null;default:0;index"`
	GeoStatus          int     `gorm:"not null;default:0;index:idx_status_geo_status,priority:2"`
	GeoSource          string  `gorm:"size:32;not null;default:''"`
	GeoScore           int     `gorm:"not null;default:0"`
	GeoLevel           string  `gorm:"size:64;not null;default:''"`
	GeoAttempts        int     `gorm:"not null;default:0"`
	GeoError           string  `gorm:"size:255;not null;default:''"`
	GeoUpdatedAt       *time.Time
	DuplicateOfVenueId *int64    `gorm:"default:null"`
	RejectReason       string    `gorm:"size:255;not null;default:''"`
	CreatedAt          time.Time `gorm:"autoCreateTime"`
}

func (favoriteVenueRewardVenueSchema) TableName() string {
	return "venues"
}

type favoriteVenueRewardConfigSchema struct {
	Id                int64  `gorm:"primarykey"`
	ActivityKey       string `gorm:"size:64;not null;uniqueIndex"`
	Enabled           bool   `gorm:"not null;default:false"`
	PopupEnabled      bool   `gorm:"not null;default:false"`
	RewardDays        int    `gorm:"not null;default:30"`
	NewUserWindowDays int    `gorm:"not null;default:7"`
	StartAt           *time.Time
	EndAt             *time.Time
	CreatedAt         time.Time `gorm:"autoCreateTime"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime"`
}

func (favoriteVenueRewardConfigSchema) TableName() string {
	return "favorite_venue_reward_configs"
}

type favoriteVenueRewardRecordSchema struct {
	Id                    int64  `gorm:"primarykey"`
	ActivityKey           string `gorm:"size:64;not null;uniqueIndex:uniq_activity_user,priority:1;uniqueIndex:uniq_activity_venue,priority:1"`
	UserId                int64  `gorm:"not null;uniqueIndex:uniq_activity_user,priority:2;index"`
	VenueId               int64  `gorm:"not null;uniqueIndex:uniq_activity_venue,priority:2;index"`
	RewardDays            int    `gorm:"not null;default:30"`
	MemberExpiresAtBefore *time.Time
	MemberExpiresAtAfter  time.Time `gorm:"not null"`
	GrantedAt             time.Time `gorm:"not null"`
	CreatedAt             time.Time `gorm:"autoCreateTime"`
}

func (favoriteVenueRewardRecordSchema) TableName() string {
	return "favorite_venue_reward_records"
}

// PrepareFavoriteVenueRewardSchema bootstraps sqlite-backed tests for the favorite venue reward workflow.
func PrepareFavoriteVenueRewardSchema(db *gorm.DB) error {
	return db.AutoMigrate(
		&favoriteVenueRewardUserSchema{},
		&favoriteVenueRewardVenueSchema{},
		&favoriteVenueRewardConfigSchema{},
		&favoriteVenueRewardRecordSchema{},
	)
}
