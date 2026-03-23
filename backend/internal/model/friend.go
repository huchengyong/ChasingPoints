package model

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Friend struct {
	Id        int64     `gorm:"primarykey"`
	UserId    int64     `gorm:"not null;index:idx_user_friend,unique"`
	FriendId  int64     `gorm:"not null;index:idx_user_friend,unique"`
	Status    int       `gorm:"not null;default:1"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (Friend) TableName() string {
	return "friends"
}

type FriendRequest struct {
	Id         int64     `gorm:"primarykey"`
	FromUserId int64     `gorm:"not null;index"`
	ToUserId   int64     `gorm:"not null;index"`
	Status     int       `gorm:"not null;default:0"`
	Message    string    `gorm:"size:200"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

func (FriendRequest) TableName() string {
	return "friend_requests"
}

type FriendModel struct {
	db *gorm.DB
}

func NewFriendModel(db *gorm.DB) *FriendModel {
	return &FriendModel{db: db}
}

func (m *FriendModel) AreFriends(userId1, userId2 int64) (bool, error) {
	var count int64
	err := m.db.Model(&Friend{}).
		Where("status = 1").
		Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)", userId1, userId2, userId2, userId1).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (m *FriendModel) GetFriendCount(userId int64) (int64, error) {
	var count int64
	err := m.db.Model(&Friend{}).
		Where("user_id = ? AND status = 1", userId).
		Count(&count).Error
	return count, err
}

func (m *FriendModel) GetFriendList(userId int64, page, pageSize int) ([]Friend, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	query := m.db.Model(&Friend{}).Where("user_id = ? AND status = 1", userId)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []Friend
	err := query.Order("created_at DESC, id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (m *FriendModel) AddFriend(userId, friendId int64) error {
	if userId == friendId {
		return fmt.Errorf("cannot add self as friend")
	}

	return m.db.Transaction(func(tx *gorm.DB) error {
		pairs := []Friend{
			{UserId: userId, FriendId: friendId, Status: 1},
			{UserId: friendId, FriendId: userId, Status: 1},
		}
		for _, item := range pairs {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&item).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (m *FriendModel) DeleteFriend(userId, friendId int64) error {
	return m.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)", userId, friendId, friendId, userId).Delete(&Friend{}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (m *FriendModel) SendRequest(fromUserId, toUserId int64, message string) error {
	request := &FriendRequest{
		FromUserId: fromUserId,
		ToUserId:   toUserId,
		Status:     0,
		Message:    message,
	}
	return m.db.Create(request).Error
}

func (m *FriendModel) GetPendingRequests(userId int64) ([]FriendRequest, error) {
	var list []FriendRequest
	err := m.db.Where("to_user_id = ? AND status = 0", userId).
		Order("created_at DESC, id DESC").
		Find(&list).Error
	return list, err
}

func (m *FriendModel) AcceptRequest(requestId, userId int64) error {
	result := m.db.Model(&FriendRequest{}).
		Where("id = ? AND to_user_id = ? AND status = 0", requestId, userId).
		Update("status", 1)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (m *FriendModel) RejectRequest(requestId, userId int64) error {
	result := m.db.Model(&FriendRequest{}).
		Where("id = ? AND to_user_id = ? AND status = 0", requestId, userId).
		Update("status", 2)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (m *FriendModel) GetRequestById(requestId int64) (*FriendRequest, error) {
	var request FriendRequest
	err := m.db.First(&request, requestId).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (m *FriendModel) HasPendingRequest(userId1, userId2 int64) (bool, error) {
	var count int64
	err := m.db.Model(&FriendRequest{}).
		Where("status = 0").
		Where("(from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?)", userId1, userId2, userId2, userId1).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (m *FriendModel) SearchUsers(keyword string, limit int) ([]User, error) {
	if keyword == "" {
		return []User{}, nil
	}
	if limit <= 0 {
		limit = 20
	}

	var list []User
	err := m.db.Model(&User{}).
		Where("status = 1").
		Where("nickname LIKE ? OR phone = ?", "%"+keyword+"%", keyword).
		Order("id DESC").
		Limit(limit).
		Find(&list).Error
	return list, err
}
