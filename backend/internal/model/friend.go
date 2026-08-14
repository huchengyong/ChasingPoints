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

type FriendBlacklist struct {
	Id            int64     `gorm:"primarykey"`
	UserId        int64     `gorm:"not null;index:idx_user_blocked,unique"`
	BlockedUserId int64     `gorm:"not null;index:idx_user_blocked,unique;index:idx_blocked_user"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
}

func (FriendBlacklist) TableName() string {
	return "friend_blacklists"
}

type FriendListProfile struct {
	Id        int64     `gorm:"column:id"`
	UserId    int64     `gorm:"column:user_id"`
	Nickname  string    `gorm:"column:nickname"`
	Avatar    string    `gorm:"column:avatar"`
	RankLevel int       `gorm:"column:rank_level"`
	RankName  string    `gorm:"column:rank_name"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

type FriendRequestProfile struct {
	Id         int64     `gorm:"column:id"`
	FromUserId int64     `gorm:"column:from_user_id"`
	Nickname   string    `gorm:"column:nickname"`
	Avatar     string    `gorm:"column:avatar"`
	Message    string    `gorm:"column:message"`
	Status     int       `gorm:"column:status"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

type SearchUserProfile struct {
	UserId            int64  `gorm:"column:user_id"`
	Nickname          string `gorm:"column:nickname"`
	Avatar            string `gorm:"column:avatar"`
	RankName          string `gorm:"column:rank_name"`
	IsFriend          bool   `gorm:"column:is_friend"`
	HasPendingRequest bool   `gorm:"column:has_pending_request"`
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

func (m *FriendModel) ListFriendProfiles(userId int64, limit int) ([]FriendListProfile, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	var list []FriendListProfile
	err := m.db.Table("friends AS f").
		Select(`f.id, f.friend_id AS user_id, f.created_at, u.nickname, u.avatar,
			COALESCE(r.rank_level, 1) AS rank_level, COALESCE(c.name, '') AS rank_name`).
		Joins("JOIN users AS u ON u.id = f.friend_id").
		Joins("LEFT JOIN user_ranking AS r ON r.user_id = f.friend_id AND r.game_type = ?", defaultRankingGameType).
		Joins("LEFT JOIN rank_config AS c ON c.level = COALESCE(r.rank_level, 1)").
		Where("f.user_id = ? AND f.status = 1", userId).
		Order("f.created_at DESC, f.id DESC").
		Limit(limit).
		Scan(&list).Error
	return list, err
}

func (m *FriendModel) GetFriendListWithProfiles(userId int64, page, pageSize int) ([]FriendListProfile, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int64
	if err := m.db.Model(&Friend{}).Where("user_id = ? AND status = 1", userId).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []FriendListProfile
	err := m.db.Table("friends AS f").
		Select(`f.id, f.friend_id AS user_id, f.created_at, u.nickname, u.avatar,
			COALESCE(r.rank_level, 1) AS rank_level, COALESCE(c.name, '') AS rank_name`).
		Joins("JOIN users AS u ON u.id = f.friend_id").
		Joins("LEFT JOIN user_ranking AS r ON r.user_id = f.friend_id AND r.game_type = ?", defaultRankingGameType).
		Joins("LEFT JOIN rank_config AS c ON c.level = COALESCE(r.rank_level, 1)").
		Where("f.user_id = ? AND f.status = 1", userId).
		Order("f.created_at DESC, f.id DESC").
		Offset(offset).
		Limit(pageSize).
		Scan(&list).Error
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

func (m *FriendModel) SendRequest(fromUserId, toUserId int64, message string) (*FriendRequest, error) {
	request := &FriendRequest{
		FromUserId: fromUserId,
		ToUserId:   toUserId,
		Status:     0,
		Message:    message,
	}
	if err := m.db.Create(request).Error; err != nil {
		return nil, err
	}
	return request, nil
}

func (m *FriendModel) HasBlacklistRelation(userId1, userId2 int64) (bool, error) {
	var count int64
	err := m.db.Model(&FriendBlacklist{}).
		Where("(user_id = ? AND blocked_user_id = ?) OR (user_id = ? AND blocked_user_id = ?)", userId1, userId2, userId2, userId1).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (m *FriendModel) BlacklistFriend(userId, blockedUserId int64) error {
	if userId == blockedUserId {
		return fmt.Errorf("cannot blacklist self")
	}

	return m.db.Transaction(func(tx *gorm.DB) error {
		record := FriendBlacklist{
			UserId:        userId,
			BlockedUserId: blockedUserId,
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&record).Error; err != nil {
			return err
		}
		if err := tx.Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)", userId, blockedUserId, blockedUserId, userId).Delete(&Friend{}).Error; err != nil {
			return err
		}
		if err := tx.Where("status = 0").Where("(from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?)", userId, blockedUserId, blockedUserId, userId).Delete(&FriendRequest{}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (m *FriendModel) GetPendingRequestCount(userId int64) (int64, error) {
	var count int64
	err := m.db.Model(&FriendRequest{}).
		Where("to_user_id = ? AND status = 0", userId).
		Count(&count).Error
	return count, err
}

func (m *FriendModel) GetPendingRequests(userId int64) ([]FriendRequest, error) {
	var list []FriendRequest
	err := m.db.Where("to_user_id = ? AND status = 0", userId).
		Order("created_at DESC, id DESC").
		Find(&list).Error
	return list, err
}

func (m *FriendModel) GetPendingRequestPageWithProfiles(userId int64, page, pageSize int) ([]FriendRequestProfile, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	var total int64
	if err := m.db.Model(&FriendRequest{}).Where("to_user_id = ? AND status = 0", userId).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []FriendRequestProfile
	err := m.db.Table("friend_requests AS r").
		Select("r.id, r.from_user_id, r.message, r.status, r.created_at, u.nickname, u.avatar").
		Joins("JOIN users AS u ON u.id = r.from_user_id").
		Where("r.to_user_id = ? AND r.status = 0", userId).
		Order("r.created_at DESC, r.id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&list).Error
	return list, total, err
}

func (m *FriendModel) GetPendingRequestsBetweenUsers(userId1, userId2 int64) ([]FriendRequest, error) {
	var list []FriendRequest
	err := m.db.Where("status = 0").
		Where("(from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?)", userId1, userId2, userId2, userId1).
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

func (m *FriendModel) SearchUsers(searcherId int64, keyword string, limit int) ([]User, error) {
	if keyword == "" {
		return []User{}, nil
	}
	if limit <= 0 {
		limit = 20
	}

	var list []User
	err := m.db.Model(&User{}).
		Where("status = 1").
		Where("id <> ?", searcherId).
		Where("nickname LIKE ? OR phone = ?", "%"+keyword+"%", keyword).
		Where(`NOT EXISTS (
			SELECT 1
			FROM friend_blacklists
			WHERE (user_id = ? AND blocked_user_id = users.id)
				OR (user_id = users.id AND blocked_user_id = ?)
		)`, searcherId, searcherId).
		Order("id DESC").
		Limit(limit).
		Find(&list).Error
	return list, err
}

func (m *FriendModel) SearchUserProfiles(searcherId int64, keyword string, limit int) ([]SearchUserProfile, error) {
	if keyword == "" {
		return []SearchUserProfile{}, nil
	}
	if limit <= 0 {
		limit = 20
	}
	var list []SearchUserProfile
	err := m.db.Table("users AS u").
		Select(`u.id AS user_id, u.nickname, u.avatar, COALESCE(c.name, '') AS rank_name,
			CASE WHEN EXISTS (SELECT 1 FROM friends AS f WHERE f.status = 1 AND ((f.user_id = ? AND f.friend_id = u.id) OR (f.user_id = u.id AND f.friend_id = ?))) THEN 1 ELSE 0 END AS is_friend,
			CASE WHEN EXISTS (SELECT 1 FROM friend_requests AS p WHERE p.status = 0 AND ((p.from_user_id = ? AND p.to_user_id = u.id) OR (p.from_user_id = u.id AND p.to_user_id = ?))) THEN 1 ELSE 0 END AS has_pending_request`,
			searcherId, searcherId, searcherId, searcherId).
		Joins("LEFT JOIN user_ranking AS r ON r.user_id = u.id AND r.game_type = ?", defaultRankingGameType).
		Joins("LEFT JOIN rank_config AS c ON c.level = COALESCE(r.rank_level, 1)").
		Where("u.status = 1 AND u.id <> ?", searcherId).
		Where("u.nickname LIKE ? OR u.phone = ?", "%"+keyword+"%", keyword).
		Where(`NOT EXISTS (
			SELECT 1
			FROM friend_blacklists
			WHERE (user_id = ? AND blocked_user_id = u.id)
				OR (user_id = u.id AND blocked_user_id = ?)
		)`, searcherId, searcherId).
		Order("u.id DESC").
		Limit(limit).
		Scan(&list).Error
	return list, err
}
