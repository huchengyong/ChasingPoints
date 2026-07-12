package model

import (
	"time"

	"gorm.io/gorm"
)

type UserOauth struct {
	Id        int64     `gorm:"primarykey"`
	UserId    int64     `gorm:"not null;index"`
	Provider  string    `gorm:"size:20;not null"`
	OpenId    string    `gorm:"size:128;not null"`
	UnionId   *string   `gorm:"size:128"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (UserOauth) TableName() string {
	return "user_oauth"
}

type UserOauthModel struct {
	db *gorm.DB
}

func NewUserOauthModel(db *gorm.DB) *UserOauthModel {
	return &UserOauthModel{db: db}
}

// FindByProviderAndOpenId 根据 provider 和 openId 查找
func (m *UserOauthModel) FindByProviderAndOpenId(provider, openId string) (*UserOauth, error) {
	var oauth UserOauth
	err := m.db.Where("provider = ? AND open_id = ?", provider, openId).First(&oauth).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &oauth, err
}

// FindByUserId 根据用户ID查找
func (m *UserOauthModel) FindByUserId(userId int64) ([]UserOauth, error) {
	var oauths []UserOauth
	err := m.db.Where("user_id = ?", userId).Find(&oauths).Error
	return oauths, err
}

// Create 创建 OAuth 关联
func (m *UserOauthModel) Create(oauth *UserOauth) error {
	return m.CreateWithTx(nil, oauth)
}

func (m *UserOauthModel) CreateWithTx(tx *gorm.DB, oauth *UserOauth) error {
	db := m.db
	if tx != nil {
		db = tx
	}
	return db.Create(oauth).Error
}

// UpdateUserId 更新 OAuth 记录的用户ID（用于账号合并）
func (m *UserOauthModel) UpdateUserId(fromUserId, toUserId int64) error {
	return m.UpdateUserIdWithTx(nil, fromUserId, toUserId)
}

func (m *UserOauthModel) UpdateUserIdWithTx(tx *gorm.DB, fromUserId, toUserId int64) error {
	db := m.db
	if tx != nil {
		db = tx
	}
	return db.Model(&UserOauth{}).
		Where("user_id = ?", fromUserId).
		Update("user_id", toUserId).Error
}
