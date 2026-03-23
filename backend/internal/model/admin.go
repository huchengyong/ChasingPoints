package model

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Admin 管理员模型
type Admin struct {
	Id          uint64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Email       string     `gorm:"column:email;uniqueIndex:uk_email;size:255;not null" json:"email"`
	Password    string     `gorm:"column:password;size:255;not null" json:"-"` // 密码不序列化
	Nickname    string     `gorm:"column:nickname;size:100;not null;default:管理员" json:"nickname"`
	Avatar      string     `gorm:"column:avatar;size:500;default:''" json:"avatar"`
	Role        string     `gorm:"column:role;size:50;not null;default:admin" json:"role"`
	Status      int8       `gorm:"column:status;not null;default:1" json:"status"` // 0-禁用 1-启用
	LastLoginAt *time.Time `gorm:"column:last_login_at" json:"last_login_at"`
	CreatedAt   time.Time  `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

func (Admin) TableName() string {
	return "admins"
}

// AdminModel 管理员模型操作
type AdminModel struct {
	db *gorm.DB
}

func NewAdminModel(db *gorm.DB) *AdminModel {
	return &AdminModel{db: db}
}

// FindByEmail 根据邮箱查找管理员
func (m *AdminModel) FindByEmail(ctx context.Context, email string) (*Admin, error) {
	var admin Admin
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	err := m.db.WithContext(ctx).Where("email = ?", normalizedEmail).First(&admin).Error
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

// FindById 根据ID查找管理员
func (m *AdminModel) FindById(ctx context.Context, id uint64) (*Admin, error) {
	var admin Admin
	err := m.db.WithContext(ctx).First(&admin, id).Error
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

// Create 创建管理员
func (m *AdminModel) Create(ctx context.Context, admin *Admin) error {
	return m.db.WithContext(ctx).Create(admin).Error
}

// UpdateLastLoginAt 更新最后登录时间
func (m *AdminModel) UpdateLastLoginAt(ctx context.Context, id uint64) error {
	now := time.Now()
	return m.db.WithContext(ctx).Model(&Admin{}).Where("id = ?", id).Update("last_login_at", now).Error
}

// UpdatePassword 更新管理员密码
func (m *AdminModel) UpdatePassword(ctx context.Context, id uint64, hashedPassword string) error {
	return m.db.WithContext(ctx).
		Model(&Admin{}).
		Where("id = ?", id).
		Update("password", hashedPassword).
		Error
}

// Count 获取管理员总数
func (m *AdminModel) Count(ctx context.Context) (int64, error) {
	var count int64
	err := m.db.WithContext(ctx).Model(&Admin{}).Count(&count).Error
	return count, err
}
