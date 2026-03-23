package model

import (
	"time"

	"gorm.io/gorm"
)

type Notification struct {
	Id        int64     `gorm:"primarykey"`
	UserId    int64     `gorm:"not null;index"`
	Type      string    `gorm:"size:30;not null"`
	Title     string    `gorm:"size:200;not null"`
	Content   string    `gorm:"size:500"`
	Data      *string   `gorm:"type:json"`
	IsRead    int       `gorm:"not null;default:0"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (Notification) TableName() string {
	return "notifications"
}

type NotificationModel struct {
	db *gorm.DB
}

func NewNotificationModel(db *gorm.DB) *NotificationModel {
	return &NotificationModel{db: db}
}

func (m *NotificationModel) FindByUserId(userId int64, page, pageSize int, notificationType string) ([]Notification, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	db := m.db.Model(&Notification{}).Where("user_id = ?", userId)
	if notificationType != "" {
		db = db.Where("type = ?", notificationType)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []Notification
	err := db.Order("created_at DESC, id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (m *NotificationModel) MarkAsRead(userId, notificationId int64) error {
	result := m.db.Model(&Notification{}).
		Where("user_id = ? AND id = ?", userId, notificationId).
		Update("is_read", 1)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (m *NotificationModel) MarkAllAsRead(userId int64) error {
	return m.db.Model(&Notification{}).
		Where("user_id = ? AND is_read = 0", userId).
		Update("is_read", 1).Error
}

func (m *NotificationModel) GetUnreadCount(userId int64) (int64, error) {
	var count int64
	err := m.db.Model(&Notification{}).
		Where("user_id = ? AND is_read = 0", userId).
		Count(&count).Error
	return count, err
}

func (m *NotificationModel) Delete(userId, notificationId int64) error {
	result := m.db.Where("user_id = ? AND id = ?", userId, notificationId).Delete(&Notification{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (m *NotificationModel) Create(notification *Notification) error {
	return m.db.Create(notification).Error
}
