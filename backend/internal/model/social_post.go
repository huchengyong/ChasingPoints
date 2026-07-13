package model

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	SocialPostStatusPublished = 1
	SocialPostStatusPending   = 2
	SocialPostStatusRejected  = 3
)

type SocialPost struct {
	Id            int64     `gorm:"primarykey"`
	UserId        int64     `gorm:"not null;index"`
	Content       string    `gorm:"size:2000;not null"`
	Images        *string   `gorm:"type:json"`
	PostType      int       `gorm:"not null;default:3"`
	MatchId       *int64    `gorm:"index"`
	LikesCount    int       `gorm:"not null;default:0"`
	CommentsCount int       `gorm:"not null;default:0"`
	Status        int       `gorm:"not null;default:2;index"`
	RejectReason  string    `gorm:"size:255;not null;default:''"`
	ReviewedAt    *time.Time
	ReviewedBy    *int64
	CreatedAt     time.Time `gorm:"autoCreateTime"`
}

func (SocialPost) TableName() string { return "social_posts" }

type SocialPostLike struct {
	Id        int64     `gorm:"primarykey"`
	PostId    int64     `gorm:"not null;uniqueIndex:idx_post_user"`
	UserId    int64     `gorm:"not null;uniqueIndex:idx_post_user"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (SocialPostLike) TableName() string { return "social_post_likes" }

type SocialPostComment struct {
	Id        int64     `gorm:"primarykey"`
	PostId    int64     `gorm:"not null;index"`
	UserId    int64     `gorm:"not null"`
	Content   string    `gorm:"size:500;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (SocialPostComment) TableName() string { return "social_post_comments" }

type SocialPostModel struct {
	db *gorm.DB
}

func NewSocialPostModel(db *gorm.DB) *SocialPostModel {
	return &SocialPostModel{db: db}
}

func (m *SocialPostModel) Create(post *SocialPost) error {
	return m.db.Create(post).Error
}

func (m *SocialPostModel) FindById(id int64) (*SocialPost, error) {
	var post SocialPost
	err := m.db.First(&post, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &post, err
}

func (m *SocialPostModel) FindPublicPosts(page, pageSize int) ([]SocialPost, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	db := m.db.Model(&SocialPost{}).Where("status = ?", SocialPostStatusPublished)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []SocialPost
	err := db.Order("created_at DESC, id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (m *SocialPostModel) FindPostsByUserIds(userIds []int64, page, pageSize int) ([]SocialPost, int64, error) {
	if len(userIds) == 0 {
		return []SocialPost{}, 0, nil
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	db := m.db.Model(&SocialPost{}).Where("user_id IN ? AND status = ?", userIds, SocialPostStatusPublished)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []SocialPost
	err := db.Order("created_at DESC, id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (m *SocialPostModel) FindPostsByUser(userId int64, page, pageSize int) ([]SocialPost, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	db := m.db.Model(&SocialPost{}).Where("user_id = ?", userId)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []SocialPost
	err := db.Order("created_at DESC, id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (m *SocialPostModel) FindListForAdmin(page, pageSize, status int) ([]SocialPost, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	db := m.db.Model(&SocialPost{})
	if status >= 0 {
		db = db.Where("status = ?", status)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []SocialPost
	err := db.Order("created_at DESC, id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (m *SocialPostModel) Delete(userId, postId int64) error {
	result := m.db.Where("id = ? AND user_id = ?", postId, userId).Delete(&SocialPost{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (m *SocialPostModel) AddLike(postId, userId int64) error {
	return m.db.Transaction(func(tx *gorm.DB) error {
		like := &SocialPostLike{PostId: postId, UserId: userId}
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(like)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}

		update := tx.Model(&SocialPost{}).
			Where("id = ?", postId).
			Update("likes_count", gorm.Expr("likes_count + ?", 1))
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		return nil
	})
}

func (m *SocialPostModel) UpdateReview(postId int64, status int, rejectReason string, reviewedBy int64, reviewedAt time.Time) error {
	updates := map[string]any{
		"status":        status,
		"reject_reason": rejectReason,
		"reviewed_at":   reviewedAt,
		"reviewed_by":   reviewedBy,
	}
	if status == SocialPostStatusPublished {
		updates["reject_reason"] = ""
	}
	return m.db.Model(&SocialPost{}).
		Where("id = ? AND status = ?", postId, SocialPostStatusPending).
		Updates(updates).Error
}

func (m *SocialPostModel) RemoveLike(postId, userId int64) error {
	return m.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("post_id = ? AND user_id = ?", postId, userId).Delete(&SocialPostLike{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}

		update := tx.Model(&SocialPost{}).
			Where("id = ?", postId).
			Update("likes_count", gorm.Expr("GREATEST(likes_count - 1, 0)"))
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		return nil
	})
}

func (m *SocialPostModel) HasLiked(postId, userId int64) (bool, error) {
	var count int64
	err := m.db.Model(&SocialPostLike{}).
		Where("post_id = ? AND user_id = ?", postId, userId).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (m *SocialPostModel) AddComment(comment *SocialPostComment) error {
	return m.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(comment).Error; err != nil {
			return err
		}

		update := tx.Model(&SocialPost{}).
			Where("id = ?", comment.PostId).
			Update("comments_count", gorm.Expr("comments_count + ?", 1))
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		return nil
	})
}

func (m *SocialPostModel) GetComments(postId int64, page, pageSize int) ([]SocialPostComment, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	db := m.db.Model(&SocialPostComment{}).Where("post_id = ?", postId)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []SocialPostComment
	err := db.Order("created_at DESC, id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}
