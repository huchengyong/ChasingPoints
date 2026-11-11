package model

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Follow struct {
	Id          int64     `gorm:"primarykey"`
	FollowerId  int64     `gorm:"not null;index:idx_follow,unique"`
	FollowingId int64     `gorm:"not null;index:idx_follow,unique;index:idx_following"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

func (Follow) TableName() string { return "follows" }

type FollowModel struct {
	db *gorm.DB
}

func NewFollowModel(db *gorm.DB) *FollowModel {
	return &FollowModel{db: db}
}

func (m *FollowModel) Follow(followerId, followingId int64) (bool, error) {
	follow := &Follow{FollowerId: followerId, FollowingId: followingId}
	result := m.db.Clauses(clause.OnConflict{DoNothing: true}).Create(follow)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (m *FollowModel) Unfollow(followerId, followingId int64) error {
	return m.db.Where("follower_id = ? AND following_id = ?", followerId, followingId).Delete(&Follow{}).Error
}

func (m *FollowModel) IsFollowing(followerId, followingId int64) (bool, error) {
	var count int64
	err := m.db.Model(&Follow{}).Where("follower_id = ? AND following_id = ?", followerId, followingId).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// IsFollowingBatch 一次 IN 查询返回 followerId 已关注的目标集合，避免列表逐条查询。
func (m *FollowModel) IsFollowingBatch(followerId int64, followingIds []int64) (map[int64]bool, error) {
	result := make(map[int64]bool, len(followingIds))
	if len(followingIds) == 0 {
		return result, nil
	}
	var ids []int64
	if err := m.db.Model(&Follow{}).
		Where("follower_id = ? AND following_id IN ?", followerId, followingIds).
		Pluck("following_id", &ids).Error; err != nil {
		return nil, err
	}
	for _, id := range ids {
		result[id] = true
	}
	return result, nil
}

func (m *FollowModel) GetFollowingList(userId int64, page, pageSize int) ([]Follow, int64, error) {
	if page <= 0 {
		page = 1
	}
	pageSize = normalizePageSize(pageSize)

	offset := (page - 1) * pageSize
	db := m.db.Model(&Follow{}).Where("follower_id = ?", userId)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []Follow
	err := db.Order("created_at DESC, id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (m *FollowModel) GetFollowerList(userId int64, page, pageSize int) ([]Follow, int64, error) {
	if page <= 0 {
		page = 1
	}
	pageSize = normalizePageSize(pageSize)

	offset := (page - 1) * pageSize
	db := m.db.Model(&Follow{}).Where("following_id = ?", userId)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []Follow
	err := db.Order("created_at DESC, id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}
