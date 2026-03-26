package model

import (
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
	MemberExpiresAt *time.Time     `json:"member_expires_at"`
	CreatedAt       time.Time      `gorm:"autoCreateTime"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

func (User) TableName() string {
	return "users"
}

type UserModel struct {
	db *gorm.DB
}

func NewUserModel(db *gorm.DB) *UserModel {
	db.AutoMigrate(&User{})
	return &UserModel{db: db}
}

// FindByPhone 根据手机号查找用户
func (m *UserModel) FindByPhone(phone string) (*User, error) {
	var user User
	err := m.db.Where("phone = ?", phone).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &user, err
}

// FindById 根据ID查找用户
func (m *UserModel) FindById(id int64) (*User, error) {
	var user User
	err := m.db.First(&user, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &user, err
}

// Create 创建用户
func (m *UserModel) Create(user *User) error {
	return m.db.Create(user).Error
}

// Update 更新用户
func (m *UserModel) Update(user *User) error {
	return m.db.Save(user).Error
}

// UpdatePhone 更新用户手机号
func (m *UserModel) UpdatePhone(userId int64, phone string) error {
	return m.db.Model(&User{}).Where("id = ?", userId).Update("phone", phone).Error
}

// DeleteById 软删除用户（用于账号合并时清理旧用户）
func (m *UserModel) DeleteById(userId int64) error {
	return m.db.Delete(&User{}, userId).Error
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
