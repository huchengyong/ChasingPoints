package geocode

import (
	"context"
	"sort"
	"time"

	"chasing_points/internal/model"
)

type QuotaLimiter interface {
	CurrentUsage(ctx context.Context, accountID int64) (int64, error)
	TryReserve(ctx context.Context, accountID int64, limit int) (bool, error)
}

type AccountSelector struct {
	limiter QuotaLimiter
}

func NewAccountSelector(limiter QuotaLimiter) *AccountSelector {
	return &AccountSelector{limiter: limiter}
}

func (s *AccountSelector) Select(ctx context.Context, accounts []model.GeocodeAccount, now time.Time) (*model.GeocodeAccount, error) {
	type candidate struct {
		account model.GeocodeAccount
		usage   int64
	}

	candidates := make([]candidate, 0, len(accounts))
	for _, account := range accounts {
		if account.Status != GeocodeAccountStatusEnabled {
			continue
		}
		if account.PerMinuteLimit <= 0 {
			continue
		}
		if account.CoolDownUntil != nil && account.CoolDownUntil.After(now) {
			continue
		}

		usage, err := s.limiter.CurrentUsage(ctx, account.Id)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate{
			account: account,
			usage:   usage,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].account.Priority != candidates[j].account.Priority {
			return candidates[i].account.Priority < candidates[j].account.Priority
		}
		if candidates[i].usage != candidates[j].usage {
			return candidates[i].usage < candidates[j].usage
		}
		return candidates[i].account.Id < candidates[j].account.Id
	})

	for _, candidate := range candidates {
		ok, err := s.limiter.TryReserve(ctx, candidate.account.Id, candidate.account.PerMinuteLimit)
		if err != nil {
			return nil, err
		}
		if ok {
			account := candidate.account
			return &account, nil
		}
	}

	return nil, ErrNoAvailableAccount
}
