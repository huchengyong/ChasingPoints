package model

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type User struct {
	Id              int64          `gorm:"primarykey"`
	Phone           *string        `gorm:"uniqueIndex;size:20"`
	Nickname        string         `gorm:"size:50;not null;default:''"`
	Avatar          string         `gorm:"size:255;not null;default:''"`
	Status          int            `gorm:"not null;default:1"`
	PushToken       string         `gorm:"size:255;not null;default:''"`
	MemberExpiresAt *time.Time     `gorm:"comment:会员到期时间" json:"member_expires_at"`
	HideMatchRecord bool           `gorm:"not null;default:false" json:"hide_match_record"`
	CreatedAt       time.Time      `gorm:"autoCreateTime"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

const DeletedUserDisplayName = "已注销用户"

func (User) TableName() string {
	return "users"
}

type UserModel struct {
	db *gorm.DB
}

func NewUserModel(db *gorm.DB) *UserModel {
	return &UserModel{db: db}
}

// FindByPhone 根据手机号查找用户
func (m *UserModel) FindByPhone(phone string) (*User, error) {
	return m.FindByPhoneWithTx(nil, phone)
}

func (m *UserModel) FindByPhoneWithTx(tx *gorm.DB, phone string) (*User, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	var user User
	err := db.Where("phone = ?", phone).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &user, err
}

func (m *UserModel) FindByPhoneForUpdateWithTx(tx *gorm.DB, phone string) (*User, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	return m.FindByPhoneWithTx(db.Clauses(clause.Locking{Strength: "UPDATE"}), phone)
}

// FindById 根据ID查找用户
func (m *UserModel) FindById(id int64) (*User, error) {
	return m.FindByIdWithTx(nil, id)
}

func (m *UserModel) FindByIdWithContext(ctx context.Context, id int64) (*User, error) {
	if ctx == nil {
		return m.FindById(id)
	}
	return m.FindByIdWithTx(m.db.WithContext(ctx), id)
}

func (m *UserModel) FindByIdWithTx(tx *gorm.DB, id int64) (*User, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	var user User
	err := db.First(&user, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &user, err
}

func (m *UserModel) FindByIds(ids []int64) (map[int64]User, error) {
	return m.FindByIdsWithTx(nil, ids)
}

func (m *UserModel) FindByIdsWithTx(tx *gorm.DB, ids []int64) (map[int64]User, error) {
	users := make(map[int64]User, len(ids))
	db := m.db
	if tx != nil {
		db = tx
	}
	const batchSize = 500
	for start := 0; start < len(ids); start += batchSize {
		end := start + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		var list []User
		if err := db.Where("id IN ?", ids[start:end]).Find(&list).Error; err != nil {
			return nil, err
		}
		for _, user := range list {
			users[user.Id] = user
		}
	}
	return users, nil
}

func (m *UserModel) FindByIdForUpdateWithTx(tx *gorm.DB, id int64) (*User, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	return m.FindByIdWithTx(db.Clauses(clause.Locking{Strength: "UPDATE"}), id)
}

// FindByIdIncludingDeletedForUpdateWithTx is used by account deletion to make
// concurrent deletion requests converge on the same terminal state.
func (m *UserModel) FindByIdIncludingDeletedForUpdateWithTx(tx *gorm.DB, id int64) (*User, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	var user User
	err := db.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &user, err
}

func (m *UserModel) ListIDsAfter(afterID int64, limit int) ([]int64, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	var ids []int64
	err := m.db.Model(&User{}).Where("id > ?", afterID).Order("id ASC").Limit(limit).Pluck("id", &ids).Error
	return ids, err
}

// Create 创建用户
func (m *UserModel) Create(user *User) error {
	return m.CreateWithTx(nil, user)
}

func (m *UserModel) CreateWithTx(tx *gorm.DB, user *User) error {
	db := m.db
	if tx != nil {
		db = tx
	}
	return db.Create(user).Error
}

// Update 更新用户
func (m *UserModel) Update(user *User) error {
	return m.db.Save(user).Error
}

// UpdatePhone 更新用户手机号
func (m *UserModel) UpdatePhone(userId int64, phone string) error {
	return m.UpdatePhoneWithTx(nil, userId, phone)
}

func (m *UserModel) UpdatePhoneWithTx(tx *gorm.DB, userId int64, phone string) error {
	db := m.db
	if tx != nil {
		db = tx
	}
	return db.Model(&User{}).Where("id = ?", userId).Update("phone", phone).Error
}

// DeleteById 软删除用户（用于账号合并时清理旧用户）
func (m *UserModel) DeleteById(userId int64) error {
	return m.DeleteByIdWithTx(nil, userId)
}

func (m *UserModel) DeleteByIdWithTx(tx *gorm.DB, userId int64) error {
	db := m.db
	if tx != nil {
		db = tx
	}
	return db.Delete(&User{}, userId).Error
}

func (m *UserModel) AnonymizeAndSoftDeleteWithTx(tx *gorm.DB, userID int64) error {
	db := m.db
	if tx != nil {
		db = tx
	}
	if db == nil {
		return errors.New("user db is nil")
	}
	updated := db.Model(&User{}).
		Where("id = ? AND deleted_at IS NULL", userID).
		Updates(map[string]any{
			"phone":             nil,
			"nickname":          DeletedUserDisplayName,
			"avatar":            "",
			"status":            0,
			"push_token":        "",
			"member_expires_at": nil,
			"hide_match_record": true,
		})
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	deleted := db.Where("id = ? AND deleted_at IS NULL", userID).Delete(&User{})
	if deleted.Error != nil {
		return deleted.Error
	}
	if deleted.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (m *UserModel) Transaction(fn func(tx *gorm.DB) error) error {
	return m.db.Transaction(fn)
}

func (m *UserModel) TransactionWithContext(ctx context.Context, fn func(tx *gorm.DB) error) error {
	if m == nil || m.db == nil {
		return errors.New("user db is nil")
	}
	db := m.db
	if ctx != nil {
		db = db.WithContext(ctx)
	}
	return db.Transaction(fn)
}

// UpdatePushToken 更新用户推送令牌
func (m *UserModel) UpdatePushToken(userId int64, token string) error {
	return m.db.Model(&User{}).Where("id = ?", userId).Update("push_token", token).Error
}

func (m *UserModel) UpdateMemberExpiresAtWithTx(tx *gorm.DB, userId int64, expiresAt *time.Time) error {
	db := m.db
	if tx != nil {
		db = tx
	}
	if db == nil {
		return errors.New("user db is nil")
	}
	return db.Model(&User{}).Where("id = ?", userId).Update("member_expires_at", expiresAt).Error
}

func (m *UserModel) UpdateHideMatchRecord(userId int64, hidden bool) error {
	return m.db.Model(&User{}).Where("id = ?", userId).Update("hide_match_record", hidden).Error
}

// LockUsersForUpdate 按主键顺序锁定用户行，用于串行化涉及同一用户的关键事务。
func (m *UserModel) LockUsersForUpdate(tx *gorm.DB, userIds []int64) error {
	if tx == nil || len(userIds) == 0 {
		return nil
	}

	var users []User
	return tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id IN ?", userIds).
		Order("id ASC").
		Find(&users).Error
}

// FindListForAdmin 管理员获取用户列表
func (m *UserModel) FindListForAdmin(page, pageSize int, status int) ([]User, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	query := m.db.Model(&User{})
	if status >= 0 {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var users []User
	err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&users).Error
	return users, total, err
}

// UpdateStatus 更新用户状态
func (m *UserModel) UpdateStatus(userId int64, status int) error {
	return m.db.Model(&User{}).Where("id = ?", userId).Update("status", status).Error
}

// CountTotal 获取用户总数
func (m *UserModel) CountTotal() (int64, error) {
	var count int64
	err := m.db.Model(&User{}).Count(&count).Error
	return count, err
}

// CountTodayNew 获取今日新增用户数
func (m *UserModel) CountTodayNew() (int64, error) {
	var count int64
	today := time.Now().Format("2006-01-01")
	err := m.db.Model(&User{}).Where("DATE(created_at) = ?", today).Count(&count).Error
	return count, err
}

// FindRecent 获取最近注册用户
func (m *UserModel) FindRecent(limit int) ([]User, error) {
	if limit <= 0 {
		limit = 10
	}
	var users []User
	err := m.db.Order("id DESC").Limit(limit).Find(&users).Error
	return users, err
}
