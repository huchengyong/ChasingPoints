package geocode

import (
	"context"
	"errors"
	"testing"
	"time"

	"billiard_master/internal/model"
)

type fakeQuotaLimiter struct {
	usages       map[int64]int64
	reserveOK    map[int64]bool
	currentUsage error
	reserveErr   error
	reserved     []int64
}

func (f *fakeQuotaLimiter) CurrentUsage(_ context.Context, accountID int64) (int64, error) {
	if f.currentUsage != nil {
		return 0, f.currentUsage
	}
	return f.usages[accountID], nil
}

func (f *fakeQuotaLimiter) TryReserve(_ context.Context, accountID int64, _ int) (bool, error) {
	if f.reserveErr != nil {
		return false, f.reserveErr
	}
	if f.reserveOK[accountID] {
		f.reserved = append(f.reserved, accountID)
		return true, nil
	}
	return false, nil
}

func TestAccountSelectorPrefersLowestUsageAccount(t *testing.T) {
	limiter := &fakeQuotaLimiter{
		usages: map[int64]int64{
			1: 6,
			2: 1,
		},
		reserveOK: map[int64]bool{
			1: true,
			2: true,
		},
	}
	selector := NewAccountSelector(limiter)
	accounts := []model.GeocodeAccount{
		{Id: 1, Status: GeocodeAccountStatusEnabled, PerMinuteLimit: 10, Priority: 10},
		{Id: 2, Status: GeocodeAccountStatusEnabled, PerMinuteLimit: 10, Priority: 10},
	}

	account, err := selector.Select(context.Background(), accounts, time.Date(2026, 3, 14, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if account.Id != 2 {
		t.Fatalf("expected account 2 to be selected, got %d", account.Id)
	}
}

func TestAccountSelectorSkipsCoolingDownAccount(t *testing.T) {
	now := time.Date(2026, 3, 14, 9, 0, 0, 0, time.UTC)
	coolDownUntil := now.Add(30 * time.Second)
	limiter := &fakeQuotaLimiter{
		usages: map[int64]int64{
			1: 0,
			2: 0,
		},
		reserveOK: map[int64]bool{
			2: true,
		},
	}
	selector := NewAccountSelector(limiter)
	accounts := []model.GeocodeAccount{
		{Id: 1, Status: GeocodeAccountStatusEnabled, PerMinuteLimit: 10, CoolDownUntil: &coolDownUntil},
		{Id: 2, Status: GeocodeAccountStatusEnabled, PerMinuteLimit: 10},
	}

	account, err := selector.Select(context.Background(), accounts, now)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if account.Id != 2 {
		t.Fatalf("expected account 2 to be selected, got %d", account.Id)
	}
}

func TestAccountSelectorReturnsErrNoAvailableAccount(t *testing.T) {
	limiter := &fakeQuotaLimiter{
		usages: map[int64]int64{
			1: 9,
			2: 9,
		},
		reserveOK: map[int64]bool{},
	}
	selector := NewAccountSelector(limiter)
	accounts := []model.GeocodeAccount{
		{Id: 1, Status: GeocodeAccountStatusEnabled, PerMinuteLimit: 10},
		{Id: 2, Status: GeocodeAccountStatusEnabled, PerMinuteLimit: 10},
	}

	_, err := selector.Select(context.Background(), accounts, time.Date(2026, 3, 14, 9, 0, 0, 0, time.UTC))
	if !errors.Is(err, ErrNoAvailableAccount) {
		t.Fatalf("expected ErrNoAvailableAccount, got %v", err)
	}
}
