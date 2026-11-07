package testsupport

import (
	"time"

	"gorm.io/gorm"
)

// 这些结构按 migrations/*_add_gear_tables.sql 的形状声明，不依赖 internal/model，
// 这样 model 字段与迁移不一致时测试会直接暴露出来。

type gearBrandSchema struct {
	Id        int64     `gorm:"primarykey"`
	Name      string    `gorm:"size:64;not null;uniqueIndex:uniq_gear_brands_name"`
	Aliases   string    `gorm:"size:255;not null;default:''"`
	Sort      int       `gorm:"not null;default:0"`
	Status    int       `gorm:"not null;default:1"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (gearBrandSchema) TableName() string { return "gear_brands" }

type gearProductSchema struct {
	Id            int64     `gorm:"primarykey"`
	Category      string    `gorm:"size:16;not null;uniqueIndex:uniq_gear_products_identity,priority:1;index:idx_gear_products_list,priority:1"`
	BrandId       int64     `gorm:"not null;default:0;uniqueIndex:uniq_gear_products_identity,priority:2;index:idx_gear_products_list,priority:3"`
	Name          string    `gorm:"size:128;not null;uniqueIndex:uniq_gear_products_identity,priority:3"`
	House         bool      `gorm:"not null;default:false"`
	CueStyle      string    `gorm:"size:16;not null;default:''"`
	Roles         string    `gorm:"size:32;not null;default:''"`
	LengthMm      int       `gorm:"not null;default:0"`
	WeightG       int       `gorm:"not null;default:0"`
	TipDiameterMm float64   `gorm:"type:decimal(5,2);not null;default:0"`
	ImagePath     string    `gorm:"size:255;not null;default:''"`
	Sort          int       `gorm:"not null;default:0;index:idx_gear_products_list,priority:4"`
	Status        int       `gorm:"not null;default:1;index:idx_gear_products_list,priority:2"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}

func (gearProductSchema) TableName() string { return "gear_products" }

type userGearItemSchema struct {
	Id              int64     `gorm:"primarykey"`
	UserId          int64     `gorm:"not null;index:idx_user_gear_items_user_status,priority:1"`
	Category        string    `gorm:"size:16;not null"`
	ProductId       int64     `gorm:"not null;default:0;index:idx_user_gear_items_pending,priority:1"`
	CustomBrandId   int64     `gorm:"not null;default:0"`
	CustomBrandName string    `gorm:"size:32;not null;default:''"`
	CustomModelName string    `gorm:"size:64;not null;default:''"`
	CustomKey       string    `gorm:"size:255;not null;default:'';index:idx_user_gear_items_pending,priority:2"`
	CueStyle        string    `gorm:"size:16;not null;default:''"`
	Roles           string    `gorm:"size:32;not null;default:''"`
	TipProductId    int64     `gorm:"not null;default:0"`
	ChalkProductId  int64     `gorm:"not null;default:0"`
	Status          int       `gorm:"not null;default:1;index:idx_user_gear_items_user_status,priority:2"`
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
}

func (userGearItemSchema) TableName() string { return "user_gear_items" }

type userGearDefaultSchema struct {
	Id        int64     `gorm:"primarykey"`
	UserId    int64     `gorm:"not null;uniqueIndex:uniq_user_gear_defaults,priority:1"`
	GameType  int       `gorm:"not null;uniqueIndex:uniq_user_gear_defaults,priority:2"`
	Slot      string    `gorm:"size:8;not null;uniqueIndex:uniq_user_gear_defaults,priority:3"`
	ItemId    int64     `gorm:"not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (userGearDefaultSchema) TableName() string { return "user_gear_defaults" }

type matchGearSnapshotSchema struct {
	Id             int64     `gorm:"primarykey"`
	MatchId        int64     `gorm:"not null;uniqueIndex:uniq_match_gear_snapshots,priority:1;index:idx_match_gear_snapshots_user,priority:2"`
	UserId         int64     `gorm:"not null;uniqueIndex:uniq_match_gear_snapshots,priority:2;index:idx_match_gear_snapshots_user,priority:1"`
	CueItemId      int64     `gorm:"not null;default:0"`
	TipProductId   int64     `gorm:"not null;default:0"`
	ChalkProductId int64     `gorm:"not null;default:0"`
	BreakItemId    int64     `gorm:"not null;default:0"`
	Source         string    `gorm:"size:16;not null;default:'auto'"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
}

func (matchGearSnapshotSchema) TableName() string { return "match_gear_snapshots" }

// PrepareGearSchema 创建装备相关的五张测试表，并预置与迁移一致的球房公杆、球房巧克。
func PrepareGearSchema(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&gearBrandSchema{},
		&gearProductSchema{},
		&userGearItemSchema{},
		&userGearDefaultSchema{},
		&matchGearSnapshotSchema{},
	); err != nil {
		return err
	}
	return db.Exec("INSERT OR IGNORE INTO gear_products (category, brand_id, name, house, sort, status) VALUES " +
		"('cue', 0, '球房公杆', 1, 0, 1), ('chalk', 0, '球房巧克', 1, 0, 1)").Error
}
