package payment

import (
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type memberSubscriptionUserSchema struct {
	Id              int64          `gorm:"primarykey"`
	Phone           *string        `gorm:"uniqueIndex;size:20"`
	Nickname        string         `gorm:"size:50;not null;default:''"`
	Avatar          string         `gorm:"size:255;not null;default:''"`
	Status          int            `gorm:"not null;default:1"`
	PushToken       string         `gorm:"size:255;not null;default:''"`
	MemberExpiresAt *time.Time     `gorm:"comment:会员到期时间"`
	HideMatchRecord bool           `gorm:"not null;default:false"`
	CreatedAt       time.Time      `gorm:"autoCreateTime"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

func (memberSubscriptionUserSchema) TableName() string {
	return "users"
}

type memberSubscriptionOrderSchema struct {
	Id                    int64      `gorm:"primarykey"`
	OrderNo               string     `gorm:"size:64;not null;uniqueIndex"`
	UserId                int64      `gorm:"not null;index"`
	PlanCode              string     `gorm:"size:64;not null"`
	PlanName              string     `gorm:"size:64;not null"`
	DurationDays          int        `gorm:"not null;default:30"`
	AmountFen             int        `gorm:"not null"`
	PayChannel            string     `gorm:"size:32;not null"`
	Status                string     `gorm:"size:32;not null;default:'pending';index"`
	ThirdPartyOrderNo     string     `gorm:"size:128;not null;default:''"`
	PaidAt                *time.Time `gorm:"comment:支付时间"`
	MemberExpiresAtBefore *time.Time `gorm:"comment:发放前会员到期时间"`
	MemberExpiresAtAfter  *time.Time `gorm:"comment:发放后会员到期时间"`
	CreatedAt             time.Time  `gorm:"autoCreateTime"`
	UpdatedAt             time.Time  `gorm:"autoUpdateTime"`
}

func (memberSubscriptionOrderSchema) TableName() string {
	return "member_subscription_orders"
}

func newMemberSubscriptionServiceTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&memberSubscriptionUserSchema{}, &memberSubscriptionOrderSchema{}); err != nil {
		t.Fatalf("prepare member subscription schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:                           db,
		UserModel:                    model.NewUserModel(db),
		MemberSubscriptionOrderModel: model.NewMemberSubscriptionOrderModel(db),
	}
}

func seedMemberSubscriptionUser(t *testing.T, svcCtx *svc.ServiceContext, user *model.User) {
	t.Helper()

	if err := svcCtx.UserModel.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
}

func seedMemberSubscriptionOrder(t *testing.T, svcCtx *svc.ServiceContext, order *model.MemberSubscriptionOrder) {
	t.Helper()

	if err := svcCtx.MemberSubscriptionOrderModel.Create(order); err != nil {
		t.Fatalf("create order: %v", err)
	}
}

func TestMemberSubscriptionServiceMarksPaidFromCurrentTimeWhenMemberExpired(t *testing.T) {
	svcCtx := newMemberSubscriptionServiceTestSvc(t)
	seedMemberSubscriptionUser(t, svcCtx, &model.User{
		Id:       301,
		Nickname: "月卡用户",
	})
	seedMemberSubscriptionOrder(t, svcCtx, &model.MemberSubscriptionOrder{
		OrderNo:      "MSO-301",
		UserId:       301,
		PlanCode:     "member_monthly",
		PlanName:     "月卡会员",
		DurationDays: 30,
		AmountFen:    1900,
		PayChannel:   "alipay",
		Status:       model.MemberSubscriptionOrderStatusPending,
	})

	now := time.Date(2026, 3, 31, 10, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	service := NewMemberSubscriptionService(svcCtx, func() time.Time { return now })

	order, err := service.MarkOrderPaid("MSO-301", "ALI-301", 1900, now)
	if err != nil {
		t.Fatalf("mark order paid: %v", err)
	}
	if order.Status != model.MemberSubscriptionOrderStatusPaid {
		t.Fatalf("expected order status paid, got %#v", order)
	}
	expectedExpiresAt := now.Add(30 * 24 * time.Hour)
	if order.MemberExpiresAtAfter == nil || !order.MemberExpiresAtAfter.Equal(expectedExpiresAt) {
		t.Fatalf("expected member expiry %s, got %#v", expectedExpiresAt.Format(time.RFC3339), order.MemberExpiresAtAfter)
	}

	user, err := svcCtx.UserModel.FindById(301)
	if err != nil {
		t.Fatalf("find user after pay: %v", err)
	}
	if user.MemberExpiresAt == nil || !user.MemberExpiresAt.Equal(expectedExpiresAt) {
		t.Fatalf("expected persisted member expiry %s, got %#v", expectedExpiresAt.Format(time.RFC3339), user.MemberExpiresAt)
	}
}

func TestMemberSubscriptionServiceExtendsFromExistingMemberExpiry(t *testing.T) {
	svcCtx := newMemberSubscriptionServiceTestSvc(t)
	currentExpiresAt := time.Date(2026, 4, 5, 9, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	seedMemberSubscriptionUser(t, svcCtx, &model.User{
		Id:              302,
		Nickname:        "续费用户",
		MemberExpiresAt: &currentExpiresAt,
	})
	seedMemberSubscriptionOrder(t, svcCtx, &model.MemberSubscriptionOrder{
		OrderNo:      "MSO-302",
		UserId:       302,
		PlanCode:     "member_monthly",
		PlanName:     "月卡会员",
		DurationDays: 30,
		AmountFen:    1900,
		PayChannel:   "wechat",
		Status:       model.MemberSubscriptionOrderStatusPending,
	})

	now := time.Date(2026, 3, 31, 10, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	service := NewMemberSubscriptionService(svcCtx, func() time.Time { return now })

	order, err := service.MarkOrderPaid("MSO-302", "WX-302", 1900, now)
	if err != nil {
		t.Fatalf("mark renewal order paid: %v", err)
	}
	expectedExpiresAt := currentExpiresAt.Add(30 * 24 * time.Hour)
	if order.MemberExpiresAtBefore == nil || !order.MemberExpiresAtBefore.Equal(currentExpiresAt) {
		t.Fatalf("expected previous expiry %s, got %#v", currentExpiresAt.Format(time.RFC3339), order.MemberExpiresAtBefore)
	}
	if order.MemberExpiresAtAfter == nil || !order.MemberExpiresAtAfter.Equal(expectedExpiresAt) {
		t.Fatalf("expected extended expiry %s, got %#v", expectedExpiresAt.Format(time.RFC3339), order.MemberExpiresAtAfter)
	}
}

func TestMemberSubscriptionServiceDoesNotDoubleGrantOnDuplicatePaidCallback(t *testing.T) {
	svcCtx := newMemberSubscriptionServiceTestSvc(t)
	seedMemberSubscriptionUser(t, svcCtx, &model.User{
		Id:       303,
		Nickname: "幂等用户",
	})
	seedMemberSubscriptionOrder(t, svcCtx, &model.MemberSubscriptionOrder{
		OrderNo:      "MSO-303",
		UserId:       303,
		PlanCode:     "member_monthly",
		PlanName:     "月卡会员",
		DurationDays: 30,
		AmountFen:    1900,
		PayChannel:   "alipay",
		Status:       model.MemberSubscriptionOrderStatusPending,
	})

	now := time.Date(2026, 3, 31, 10, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	service := NewMemberSubscriptionService(svcCtx, func() time.Time { return now })

	first, err := service.MarkOrderPaid("MSO-303", "ALI-303", 1900, now)
	if err != nil {
		t.Fatalf("first mark paid: %v", err)
	}
	second, err := service.MarkOrderPaid("MSO-303", "ALI-303", 1900, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("second mark paid: %v", err)
	}
	if second.MemberExpiresAtAfter == nil || first.MemberExpiresAtAfter == nil || !second.MemberExpiresAtAfter.Equal(*first.MemberExpiresAtAfter) {
		t.Fatalf("expected duplicate callback to keep same expiry, got first=%#v second=%#v", first.MemberExpiresAtAfter, second.MemberExpiresAtAfter)
	}
}

func TestMemberSubscriptionServiceRejectsAmountMismatch(t *testing.T) {
	svcCtx := newMemberSubscriptionServiceTestSvc(t)
	seedMemberSubscriptionUser(t, svcCtx, &model.User{
		Id:       304,
		Nickname: "金额校验用户",
	})
	seedMemberSubscriptionOrder(t, svcCtx, &model.MemberSubscriptionOrder{
		OrderNo:      "MSO-304",
		UserId:       304,
		PlanCode:     "member_monthly",
		PlanName:     "月卡会员",
		DurationDays: 30,
		AmountFen:    1900,
		PayChannel:   "wechat",
		Status:       model.MemberSubscriptionOrderStatusPending,
	})

	now := time.Date(2026, 3, 31, 10, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	service := NewMemberSubscriptionService(svcCtx, func() time.Time { return now })

	if _, err := service.MarkOrderPaid("MSO-304", "WX-304", 1999, now); err == nil {
		t.Fatal("expected amount mismatch to return error")
	}
}
