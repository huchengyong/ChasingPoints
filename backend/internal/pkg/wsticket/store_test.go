package wsticket

import (
	"context"
	"strings"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisStoreIssuesHighEntropyTicketAndStoresOnlyHash(t *testing.T) {
	mr := miniredis.RunT(t)
	store := newTestRedisStore(mr)

	ticket, ttl, err := store.Issue(context.Background(), Claims{
		UserID: 7,
		Scope:  ScopeUser,
	}, 2*time.Minute)
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}
	if len(ticket) < 40 {
		t.Fatalf("ticket is too short for high entropy: %d", len(ticket))
	}
	if ttl > MaxTTL {
		t.Fatalf("ttl exceeds max: %s", ttl)
	}

	key := defaultKeyPrefix + hashTicket(ticket)
	if !mr.Exists(key) {
		t.Fatalf("hashed ticket key does not exist")
	}
	if mr.Exists(defaultKeyPrefix + ticket) {
		t.Fatalf("raw ticket must not be used as redis key")
	}
	value, err := mr.Get(key)
	if err != nil {
		t.Fatalf("get redis value: %v", err)
	}
	if strings.Contains(value, ticket) {
		t.Fatalf("raw ticket must not be stored in redis value")
	}
	if !strings.Contains(value, `"scope":"user"`) {
		t.Fatalf("ticket value must retain scope: %s", value)
	}
	if mr.TTL(key) > MaxTTL {
		t.Fatalf("redis ttl exceeds max: %s", mr.TTL(key))
	}
}

func TestRedisStoreConsumesTicketOnce(t *testing.T) {
	mr := miniredis.RunT(t)
	store := newTestRedisStore(mr)

	ticket, _, err := store.Issue(context.Background(), Claims{
		UserID:  9,
		Scope:   ScopeMatch,
		MatchID: 12,
	}, DefaultTTL)
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}

	claims, err := store.Consume(context.Background(), ticket, ScopeMatch)
	if err != nil {
		t.Fatalf("consume ticket: %v", err)
	}
	if claims.UserID != 9 || claims.MatchID != 12 || claims.Scope != ScopeMatch {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if _, err := store.Consume(context.Background(), ticket, ScopeMatch); err != ErrTicketNotFound {
		t.Fatalf("replay should fail with ErrTicketNotFound, got %v", err)
	}
}

func TestRedisStoreRejectsExpiredAndWrongScopeTickets(t *testing.T) {
	mr := miniredis.RunT(t)
	store := newTestRedisStore(mr)

	expiredTicket, _, err := store.Issue(context.Background(), Claims{
		UserID: 1,
		Scope:  ScopeUser,
	}, time.Second)
	if err != nil {
		t.Fatalf("issue expired candidate: %v", err)
	}
	mr.FastForward(2 * time.Second)
	if _, err := store.Consume(context.Background(), expiredTicket, ScopeUser); err != ErrTicketNotFound {
		t.Fatalf("expired ticket should be missing, got %v", err)
	}

	wrongScopeTicket, _, err := store.Issue(context.Background(), Claims{
		UserID: 2,
		Scope:  ScopeUser,
	}, DefaultTTL)
	if err != nil {
		t.Fatalf("issue wrong-scope candidate: %v", err)
	}
	if _, err := store.Consume(context.Background(), wrongScopeTicket, ScopeMatch); err != ErrTicketNotFound {
		t.Fatalf("wrong scope should be rejected, got %v", err)
	}
	if _, err := store.Consume(context.Background(), wrongScopeTicket, ScopeUser); err != ErrTicketNotFound {
		t.Fatalf("wrong-scope attempt should still consume the ticket, got %v", err)
	}
}

func TestRedisStoreValidatesClaims(t *testing.T) {
	mr := miniredis.RunT(t)
	store := newTestRedisStore(mr)

	cases := []Claims{
		{UserID: 0, Scope: ScopeUser},
		{UserID: 1, Scope: ScopeUser, MatchID: 2},
		{UserID: 1, Scope: ScopeMatch},
		{UserID: 1, Scope: "admin"},
	}
	for _, claims := range cases {
		if _, _, err := store.Issue(context.Background(), claims, DefaultTTL); err != ErrInvalidTicket {
			t.Fatalf("claims %+v should be invalid, got %v", claims, err)
		}
	}
}

func newTestRedisStore(mr *miniredis.Miniredis) *RedisStore {
	return NewRedisStore(redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	}))
}
