package testsupport

import "gorm.io/gorm"

type areaSchema struct {
	AreaId   uint32 `gorm:"primarykey;column:area_id"`
	ParentId uint32 `gorm:"column:parent_id;not null;default:0;index"`
	Name     string `gorm:"column:name;size:120;not null;default:''"`
}

func (areaSchema) TableName() string {
	return "dou_area"
}

func PrepareAreaSchema(db *gorm.DB) error {
	return db.AutoMigrate(&areaSchema{})
}
