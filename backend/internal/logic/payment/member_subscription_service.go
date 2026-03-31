package payment

import (
	"errors"
	"fmt"
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/gorm"
)

type MemberSubscriptionService struct {
	svcCtx *svc.ServiceContext
	now    func() time.Time
}

func NewMemberSubscriptionService(svcCtx *svc.ServiceContext, now func() time.Time) *MemberSubscriptionService {
	if now == nil {
		now = logicx.NowUTC8
	}
	return &MemberSubscriptionService{
		svcCtx: svcCtx,
		now:    now,
	}
}

func (s *MemberSubscriptionService) MarkOrderPaid(orderNo, thirdPartyOrderNo string, paidAmountFen int, paidAt time.Time) (*model.MemberSubscriptionOrder, error) {
	if s == nil || s.svcCtx == nil || s.svcCtx.DB == nil {
		return nil, errors.New("member subscription service db is nil")
	}

	var finalOrder *model.MemberSubscriptionOrder
	err := s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		order, err := s.svcCtx.MemberSubscriptionOrderModel.FindByOrderNoForUpdate(tx, orderNo)
		if err != nil {
			return err
		}
		if order == nil {
			return fmt.Errorf("member subscription order not found: %s", orderNo)
		}

		if order.AmountFen != paidAmountFen {
			return fmt.Errorf("paid amount mismatch: expected %d got %d", order.AmountFen, paidAmountFen)
		}

		if order.Status == model.MemberSubscriptionOrderStatusPaid {
			finalOrder = order
			return nil
		}
		if order.Status != model.MemberSubscriptionOrderStatusPending {
			return fmt.Errorf("member subscription order status invalid: %s", order.Status)
		}

		user, err := s.svcCtx.UserModel.FindById(order.UserId)
		if err != nil {
			return err
		}
		if user == nil {
			return fmt.Errorf("member subscription user not found: %d", order.UserId)
		}

		effectivePaidAt := paidAt
		if effectivePaidAt.IsZero() {
			effectivePaidAt = s.now()
		}
		baseTime := effectivePaidAt
		if user.MemberExpiresAt != nil && user.MemberExpiresAt.After(baseTime) {
			baseTime = *user.MemberExpiresAt
		}
		nextExpiresAt := baseTime.Add(time.Duration(order.DurationDays) * 24 * time.Hour)

		order.Status = model.MemberSubscriptionOrderStatusPaid
		order.ThirdPartyOrderNo = thirdPartyOrderNo
		order.PaidAt = &effectivePaidAt
		if user.MemberExpiresAt != nil {
			before := *user.MemberExpiresAt
			order.MemberExpiresAtBefore = &before
		} else {
			order.MemberExpiresAtBefore = nil
		}
		order.MemberExpiresAtAfter = &nextExpiresAt

		if err := s.svcCtx.MemberSubscriptionOrderModel.UpdateWithTx(tx, order); err != nil {
			return err
		}
		if err := s.svcCtx.UserModel.UpdateMemberExpiresAtWithTx(tx, user.Id, &nextExpiresAt); err != nil {
			return err
		}

		finalOrder = order
		return nil
	})
	if err != nil {
		return nil, err
	}
	return finalOrder, nil
}
