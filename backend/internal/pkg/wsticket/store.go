package wsticket

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	ScopeUser  = "user"
	ScopeMatch = "match"

	DefaultTTL = 30 * time.Second
	MaxTTL     = 30 * time.Second

	defaultKeyPrefix = "ws:ticket:"
	ticketBytes      = 32
)

var (
	ErrInvalidTicket  = errors.New("invalid ws ticket")
	ErrTicketNotFound = errors.New("ws ticket not found")
	ErrStoreMissing   = errors.New("ws ticket store missing")
)

type Claims struct {
	UserID  int64  `json:"user_id"`
	Scope   string `json:"scope"`
	MatchID int64  `json:"match_id,omitempty"`
}

type Store interface {
	Issue(ctx context.Context, claims Claims, ttl time.Duration) (string, time.Duration, error)
	Consume(ctx context.Context, ticket string, expectedScope string) (*Claims, error)
}

type RedisStore struct {
	client    *redis.Client
	keyPrefix string
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{
		client:    client,
		keyPrefix: defaultKeyPrefix,
	}
}

func (s *RedisStore) Issue(ctx context.Context, claims Claims, ttl time.Duration) (string, time.Duration, error) {
	if s == nil || s.client == nil {
		return "", 0, ErrStoreMissing
	}
	claims.Scope = normalizeScope(claims.Scope)
	if err := validateClaims(claims); err != nil {
		return "", 0, err
	}
	ttl = normalizeTTL(ttl)

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", 0, err
	}

	for attempt := 0; attempt < 3; attempt++ {
		ticket, err := generateTicket()
		if err != nil {
			return "", 0, err
		}
		ok, err := s.client.SetNX(ctx, s.key(ticket), payload, ttl).Result()
		if err != nil {
			return "", 0, err
		}
		if ok {
			return ticket, ttl, nil
		}
	}
	return "", 0, errors.New("failed to reserve unique ws ticket")
}

func (s *RedisStore) Consume(ctx context.Context, ticket string, expectedScope string) (*Claims, error) {
	if s == nil || s.client == nil {
		return nil, ErrStoreMissing
	}
	if strings.TrimSpace(ticket) == "" {
		return nil, ErrInvalidTicket
	}
	value, err := consumeTicketScript.Run(ctx, s.client, []string{s.key(ticket)}).Text()
	if err == redis.Nil {
		return nil, ErrTicketNotFound
	}
	if err != nil {
		return nil, err
	}

	var claims Claims
	if err := json.Unmarshal([]byte(value), &claims); err != nil {
		return nil, ErrInvalidTicket
	}
	claims.Scope = normalizeScope(claims.Scope)
	if err := validateClaims(claims); err != nil {
		return nil, err
	}
	if scope := normalizeScope(expectedScope); scope != "" && claims.Scope != scope {
		return nil, ErrTicketNotFound
	}
	return &claims, nil
}

func (s *RedisStore) key(ticket string) string {
	return s.keyPrefix + hashTicket(ticket)
}

func normalizeTTL(ttl time.Duration) time.Duration {
	if ttl <= 0 || ttl > MaxTTL {
		return DefaultTTL
	}
	return ttl
}

func normalizeScope(scope string) string {
	return strings.ToLower(strings.TrimSpace(scope))
}

func validateClaims(claims Claims) error {
	switch claims.Scope {
	case ScopeUser:
		if claims.UserID <= 0 || claims.MatchID != 0 {
			return ErrInvalidTicket
		}
	case ScopeMatch:
		if claims.UserID <= 0 || claims.MatchID <= 0 {
			return ErrInvalidTicket
		}
	default:
		return ErrInvalidTicket
	}
	return nil
}

func generateTicket() (string, error) {
	raw := make([]byte, ticketBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate ws ticket: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func hashTicket(ticket string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(ticket)))
	return hex.EncodeToString(sum[:])
}

var consumeTicketScript = redis.NewScript(`
local value = redis.call("GET", KEYS[1])
if not value then
	return nil
end
redis.call("DEL", KEYS[1])
return value
`)
