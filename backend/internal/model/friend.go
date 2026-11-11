package model

import (
	"fmt"
	"strings"
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

type FriendBlacklistProfile struct {
	UserId    int64     `gorm:"column:user_id"`
	Nickname  string    `gorm:"column:nickname"`
	Avatar    string    `gorm:"column:avatar"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

// normalizePageSize 列表上限与其他战绩列表保持一致：非法或超限时回落到 20。
func normalizePageSize(pageSize int) int {
	if pageSize < 1 || pageSize > 100 {
		return 20
	}
	return pageSize
}

type FriendModel struct {
	db *gorm.DB
}

func NewFriendModel(db *gorm.DB) *FriendModel {
	return &FriendModel{db: db}
}

func (m *FriendModel) AreFriends(userId1, userId2 int64) (bool, error) {
	return m.AreFriendsWithTx(nil, userId1, userId2)
}

func (m *FriendModel) AreFriendsWithTx(tx *gorm.DB, userId1, userId2 int64) (bool, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	var count int64
	err := db.Model(&Friend{}).
		Where("status = 1").
		Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)", userId1, userId2, userId2, userId1).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (m *FriendModel) GetFriendCount(userId int64) (int64, error) {
	return m.GetFriendCountWithTx(nil, userId)
}

func (m *FriendModel) GetFriendCountWithTx(tx *gorm.DB, userId int64) (int64, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	var count int64
	err := db.Model(&Friend{}).
		Where("user_id = ? AND status = 1", userId).
		Count(&count).Error
	return count, err
}

func (m *FriendModel) GetFriendList(userId int64, page, pageSize int) ([]Friend, int64, error) {
	if page <= 0 {
		page = 1
	}
	pageSize = normalizePageSize(pageSize)

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
	pageSize = normalizePageSize(pageSize)
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
	return m.AddFriendWithTx(nil, userId, friendId)
}

func (m *FriendModel) AddFriendWithTx(tx *gorm.DB, userId, friendId int64) error {
	if userId == friendId {
		return fmt.Errorf("cannot add self as friend")
	}

	db := m.db
	if tx != nil {
		db = tx
	}
	insert := func(tx *gorm.DB) error {
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
	}
	if tx != nil {
		return insert(tx)
	}
	return db.Transaction(insert)
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
	return m.SendRequestWithTx(nil, fromUserId, toUserId, message)
}

func (m *FriendModel) SendRequestWithTx(tx *gorm.DB, fromUserId, toUserId int64, message string) (*FriendRequest, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	request := &FriendRequest{
		FromUserId: fromUserId,
		ToUserId:   toUserId,
		Status:     0,
		Message:    message,
	}
	if err := db.Create(request).Error; err != nil {
		return nil, err
	}
	return request, nil
}

func (m *FriendModel) HasBlacklistRelation(userId1, userId2 int64) (bool, error) {
	return m.HasBlacklistRelationWithTx(nil, userId1, userId2)
}

func (m *FriendModel) HasBlacklistRelationWithTx(tx *gorm.DB, userId1, userId2 int64) (bool, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	var count int64
	err := db.Model(&FriendBlacklist{}).
		Where("(user_id = ? AND blocked_user_id = ?) OR (user_id = ? AND blocked_user_id = ?)", userId1, userId2, userId2, userId1).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetBlacklistWithProfiles 联表返回我拉黑的用户；对方已注销时仍保留该行，便于移出黑名单。
func (m *FriendModel) GetBlacklistWithProfiles(userId int64, page, pageSize int) ([]FriendBlacklistProfile, int64, error) {
	if page <= 0 {
		page = 1
	}
	pageSize = normalizePageSize(pageSize)

	var total int64
	if err := m.db.Model(&FriendBlacklist{}).Where("user_id = ?", userId).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []FriendBlacklistProfile
	err := m.db.Table("friend_blacklists AS b").
		Select(`b.blocked_user_id AS user_id, b.created_at,
			CASE WHEN u.id IS NULL OR u.deleted_at IS NOT NULL THEN ? ELSE u.nickname END AS nickname,
			COALESCE(u.avatar, '') AS avatar`, DeletedUserDisplayName).
		Joins("LEFT JOIN users AS u ON u.id = b.blocked_user_id").
		Where("b.user_id = ?", userId).
		Order("b.created_at DESC, b.id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&list).Error
	return list, total, err
}

// RemoveBlacklist 只删除“我拉黑对方”这一行，移出后不自动恢复好友关系。
func (m *FriendModel) RemoveBlacklist(userId, blockedUserId int64) (bool, error) {
	result := m.db.Where("user_id = ? AND blocked_user_id = ?", userId, blockedUserId).Delete(&FriendBlacklist{})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
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
	pageSize = normalizePageSize(pageSize)
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
	return m.AcceptRequestWithTx(nil, requestId, userId)
}

func (m *FriendModel) AcceptRequestWithTx(tx *gorm.DB, requestId, userId int64) error {
	db := m.db
	if tx != nil {
		db = tx
	}
	result := db.Model(&FriendRequest{}).
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
	return m.GetRequestByIdWithTx(nil, requestId)
}

func (m *FriendModel) GetRequestByIdWithTx(tx *gorm.DB, requestId int64) (*FriendRequest, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	var request FriendRequest
	err := db.First(&request, requestId).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &request, nil
}

// FindLatestRequest 返回同一发送方对该接收方的最近一条申请，用于拒绝冷却判断。
func (m *FriendModel) FindLatestRequestWithTx(tx *gorm.DB, fromUserId, toUserId int64) (*FriendRequest, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	var request FriendRequest
	err := db.Where("from_user_id = ? AND to_user_id = ?", fromUserId, toUserId).
		Order("id DESC").
		First(&request).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &request, nil
}

// CountRequestsSince 统计发送方从 since 起创建的好友申请数量，用于频率限制。
func (m *FriendModel) CountRequestsSinceWithTx(tx *gorm.DB, fromUserId int64, since time.Time) (int64, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	var count int64
	err := db.Model(&FriendRequest{}).
		Where("from_user_id = ? AND created_at >= ?", fromUserId, since).
		Count(&count).Error
	return count, err
}

func (m *FriendModel) HasPendingRequest(userId1, userId2 int64) (bool, error) {
	return m.HasPendingRequestWithTx(nil, userId1, userId2)
}

func (m *FriendModel) HasPendingRequestWithTx(tx *gorm.DB, userId1, userId2 int64) (bool, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	var count int64
	err := db.Model(&FriendRequest{}).
		Where("status = 0").
		Where("(from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?)", userId1, userId2, userId2, userId1).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (m *FriendModel) SearchUsers(searcherId int64, keyword string, limit int) ([]User, error) {
	keyword = normalizeUserSearchKeyword(keyword)
	if keyword == "" {
		return []User{}, nil
	}
	if limit <= 0 {
		limit = 20
	}

	var list []User
	query := m.db.Model(&User{}).
		Where("status = 1").
		Where("id <> ?", searcherId)
	query = m.applyUserSearchKeyword(query, "nickname", keyword)
	err := query.Where(`NOT EXISTS (
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
	keyword = normalizeUserSearchKeyword(keyword)
	if keyword == "" {
		return []SearchUserProfile{}, nil
	}
	if limit <= 0 {
		limit = 20
	}
	var list []SearchUserProfile
	query := m.db.Table("users AS u").
		Select(`u.id AS user_id, u.nickname, u.avatar, COALESCE(c.name, '') AS rank_name,
			CASE WHEN EXISTS (SELECT 1 FROM friends AS f WHERE f.status = 1 AND ((f.user_id = ? AND f.friend_id = u.id) OR (f.user_id = u.id AND f.friend_id = ?))) THEN 1 ELSE 0 END AS is_friend,
			CASE WHEN EXISTS (SELECT 1 FROM friend_requests AS p WHERE p.status = 0 AND ((p.from_user_id = ? AND p.to_user_id = u.id) OR (p.from_user_id = u.id AND p.to_user_id = ?))) THEN 1 ELSE 0 END AS has_pending_request`,
			searcherId, searcherId, searcherId, searcherId).
		Joins("LEFT JOIN user_ranking AS r ON r.user_id = u.id AND r.game_type = ?", defaultRankingGameType).
		Joins("LEFT JOIN rank_config AS c ON c.level = COALESCE(r.rank_level, 1)").
		Where("u.status = 1 AND u.id <> ?", searcherId)
	query = m.applyUserSearchKeyword(query, "u.nickname", keyword)
	err := query.Where(`NOT EXISTS (
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

const userSearchKeywordRuneLimit = 30

func normalizeUserSearchKeyword(keyword string) string {
	keyword = strings.TrimSpace(keyword)
	runes := []rune(keyword)
	if len(runes) > userSearchKeywordRuneLimit {
		keyword = string(runes[:userSearchKeywordRuneLimit])
	}
	return keyword
}

// applyUserSearchKeyword 统一搜索匹配：昵称模糊匹配并对通配符转义，11 位纯数字才额外精确匹配手机号。
// 转义字符用 '!'，MySQL 与 SQLite 的字符串字面量都不需要额外反斜杠。
func (m *FriendModel) applyUserSearchKeyword(query *gorm.DB, nicknameColumn, keyword string) *gorm.DB {
	pattern := "%" + strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(keyword) + "%"
	if isPhoneKeyword(keyword) {
		return query.Where("("+nicknameColumn+" LIKE ? ESCAPE '!' OR phone = ?)", pattern, keyword)
	}
	return query.Where(nicknameColumn+" LIKE ? ESCAPE '!'", pattern)
}

func isPhoneKeyword(keyword string) bool {
	if len(keyword) != 11 {
		return false
	}
	for _, char := range keyword {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}
