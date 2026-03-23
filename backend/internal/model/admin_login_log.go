package model

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// AdminLoginLog 管理员登录日志
type AdminLoginLog struct {
	Id        uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	AdminId   uint64    `gorm:"column:admin_id;not null;index:idx_admin_id" json:"admin_id"`
	Ip        string    `gorm:"column:ip;size:50;default:''" json:"ip"`
	UserAgent string    `gorm:"column:user_agent;size:500;default:''" json:"user_agent"`
	LoginAt   time.Time `gorm:"column:login_at;not null;default:CURRENT_TIMESTAMP;index:idx_login_at" json:"login_at"`
}

func (AdminLoginLog) TableName() string {
	return "admin_login_logs"
}

// AdminLoginLogModel 登录日志模型操作
type AdminLoginLogModel struct {
	db *gorm.DB
}

func NewAdminLoginLogModel(db *gorm.DB) *AdminLoginLogModel {
	return &AdminLoginLogModel{db: db}
}

// Create 创建登录日志
func (m *AdminLoginLogModel) Create(ctx context.Context, log *AdminLoginLog) error {
	return m.db.WithContext(ctx).Create(log).Error
}
