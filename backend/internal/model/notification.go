package model

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Notification struct {
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

func (m *NotificationModel) FindPageWithUnreadCount(userId int64, page, pageSize int, notificationType string) ([]Notification, int64, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize
	var list []Notification
	var total, unreadCount int64
	err := m.db.Transaction(func(tx *gorm.DB) error {
		query := tx.Model(&Notification{}).Where("user_id = ?", userId)
		if notificationType != "" {
			query = query.Where("type = ?", notificationType)
		}
		if err := query.Count(&total).Error; err != nil {
			return err
		}
		if err := query.Order("created_at DESC, id DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
			return err
		}
		return tx.Model(&Notification{}).Where("user_id = ? AND is_read = 0", userId).Count(&unreadCount).Error
	})
	return list, total, unreadCount, err
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

func (m *NotificationModel) CreateBatchIfAbsentWithTx(tx *gorm.DB, notifications []Notification, batchSize int) (int64, error) {
	if len(notifications) == 0 {
		return 0, nil
	}
	if batchSize <= 0 {
		batchSize = 500
	}
	db := m.db
	if tx != nil {
		db = tx
	}
	result := db.Clauses(notificationConflict()).CreateInBatches(&notifications, batchSize)
	return result.RowsAffected, result.Error
}

func (m *NotificationModel) CreateIfAbsent(notification *Notification) (bool, error) {
	return m.CreateIfAbsentWithTx(nil, notification)
}

func (m *NotificationModel) CreateIfAbsentWithTx(tx *gorm.DB, notification *Notification) (bool, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	result := db.Clauses(notificationConflict()).Create(notification)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func notificationConflict() clause.OnConflict {
	return clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "type"},
			{Name: "dedupe_key"},
		},
		DoNothing: true,
	}
}

func (m *NotificationModel) FindLatestUnreadByType(userId int64, notificationType string) (*Notification, error) {
	var notification Notification
	err := m.db.Where("user_id = ? AND type = ? AND is_read = 0", userId, notificationType).
		Order("created_at DESC, id DESC").
		First(&notification).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &notification, err
}

func (m *NotificationModel) DeleteFriendRequestNotification(userId, requestId int64, legacyContent string) error {
	query := m.db.Where("user_id = ? AND type = ? AND title = ?", userId, "friend_request", "收到好友申请")

	if requestId > 0 && legacyContent != "" {
		query = query.Where("(data LIKE ? OR content = ?)", fmt.Sprintf("%%\"request_id\":%d%%", requestId), legacyContent)
	} else if requestId > 0 {
		query = query.Where("data LIKE ?", fmt.Sprintf("%%\"request_id\":%d%%", requestId))
	} else if legacyContent != "" {
		query = query.Where("content = ?", legacyContent)
	}

	return query.Delete(&Notification{}).Error
}
