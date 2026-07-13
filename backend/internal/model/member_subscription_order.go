package model

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	MemberSubscriptionOrderStatusPending = "pending"
	MemberSubscriptionOrderStatusPaid    = "paid"
	MemberSubscriptionOrderStatusClosed  = "closed"
	MemberSubscriptionOrderStatusFailed  = "failed"
)

type MemberSubscriptionOrder struct {
	Id                    int64      `gorm:"primarykey" json:"id"`
	OrderNo               string     `gorm:"size:64;not null;uniqueIndex" json:"order_no"`
	UserId                int64      `gorm:"not null;index" json:"user_id"`
	PlanCode              string     `gorm:"size:64;not null" json:"plan_code"`
	PlanName              string     `gorm:"size:64;not null" json:"plan_name"`
	DurationDays          int        `gorm:"not null;default:30" json:"duration_days"`
	AmountFen             int        `gorm:"not null" json:"amount_fen"`
	PayChannel            string     `gorm:"size:32;not null" json:"pay_channel"`
	Status                string     `gorm:"size:32;not null;default:'pending';index" json:"status"`
	ThirdPartyOrderNo     string     `gorm:"size:128;not null;default:''" json:"third_party_order_no"`
	PaidAt                *time.Time `gorm:"comment:支付时间" json:"paid_at"`
	MemberExpiresAtBefore *time.Time `gorm:"comment:发放前会员到期时间" json:"member_expires_at_before"`
	MemberExpiresAtAfter  *time.Time `gorm:"comment:发放后会员到期时间" json:"member_expires_at_after"`
	CreatedAt             time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (MemberSubscriptionOrder) TableName() string {
	return "member_subscription_orders"
}

type MemberSubscriptionOrderModel struct {
	db *gorm.DB
}

func NewMemberSubscriptionOrderModel(db *gorm.DB) *MemberSubscriptionOrderModel {
	return &MemberSubscriptionOrderModel{db: db}
}

func (m *MemberSubscriptionOrderModel) Create(order *MemberSubscriptionOrder) error {
	if m == nil || m.db == nil {
		return errors.New("member subscription order db is nil")
	}
	if order == nil {
		return errors.New("member subscription order is nil")
	}
	order.OrderNo = strings.TrimSpace(order.OrderNo)
	if order.OrderNo == "" {
		return errors.New("order no is required")
	}
	if strings.TrimSpace(order.Status) == "" {
		order.Status = MemberSubscriptionOrderStatusPending
	}
	return m.db.Create(order).Error
}

func (m *MemberSubscriptionOrderModel) FindByOrderNo(orderNo string) (*MemberSubscriptionOrder, error) {
	if m == nil || m.db == nil {
		return nil, errors.New("member subscription order db is nil")
	}
	var order MemberSubscriptionOrder
	err := m.db.Where("order_no = ?", strings.TrimSpace(orderNo)).First(&order).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (m *MemberSubscriptionOrderModel) FindByOrderNoForUpdate(tx *gorm.DB, orderNo string) (*MemberSubscriptionOrder, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	if db == nil {
		return nil, errors.New("member subscription order db is nil")
	}
	var order MemberSubscriptionOrder
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("order_no = ?", strings.TrimSpace(orderNo)).
		First(&order).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (m *MemberSubscriptionOrderModel) UpdateWithTx(tx *gorm.DB, order *MemberSubscriptionOrder) error {
	db := m.db
	if tx != nil {
		db = tx
	}
	if db == nil {
		return errors.New("member subscription order db is nil")
	}
	if order == nil {
		return errors.New("member subscription order is nil")
	}
	return db.Save(order).Error
}
