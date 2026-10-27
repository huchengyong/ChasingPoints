package matchinvite

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// CodeLength 为 12 位 base62 短码，约 71 位熵。
	CodeLength = 12

	codeKeyPrefix = "match_invite_code:"
	codeAlphabet  = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	codeBytes     = 9
)

var (
	ErrCodeStoreMissing = errors.New("match invite code store is missing")
	ErrInvalidCode      = errors.New("invalid match invite code")
	ErrCodeNotFound     = errors.New("match invite code not found")

	codeLimit = new(big.Int).Exp(big.NewInt(int64(len(codeAlphabet))), big.NewInt(CodeLength), nil)
	codeBase  = big.NewInt(int64(len(codeAlphabet)))
)

// CodeStore 保存短码到签名 token 的映射，有效期跟随 token 剩余有效期。
type CodeStore struct {
	client *redis.Client
}

func NewCodeStore(client *redis.Client) *CodeStore {
	return &CodeStore{client: client}
}

func (s *CodeStore) Save(ctx context.Context, token string, ttl time.Duration) (string, error) {
	if s == nil || s.client == nil {
		return "", ErrCodeStoreMissing
	}
	if token == "" || ttl <= 0 {
		return "", ErrInvalidCode
	}

	for attempt := 0; attempt < 3; attempt++ {
		code, err := generateCode()
		if err != nil {
			return "", err
		}
		ok, err := s.client.SetNX(ctx, codeKeyPrefix+code, token, ttl).Result()
		if err != nil {
			return "", err
		}
		if ok {
			return code, nil
		}
	}
	return "", errors.New("failed to reserve unique match invite code")
}

func (s *CodeStore) Resolve(ctx context.Context, code string) (string, error) {
	if s == nil || s.client == nil {
		return "", ErrCodeStoreMissing
	}
	if !ValidCode(code) {
		return "", ErrInvalidCode
	}
	token, err := s.client.Get(ctx, codeKeyPrefix+code).Result()
	if err == redis.Nil {
		return "", ErrCodeNotFound
	}
	if err != nil {
		return "", err
	}
	return token, nil
}

func ValidCode(code string) bool {
	if len(code) != CodeLength {
		return false
	}
	for i := 0; i < len(code); i++ {
		if !isCodeChar(code[i]) {
			return false
		}
	}
	return true
}

func isCodeChar(value byte) bool {
	if value >= '0' && value <= '9' {
		return true
	}
	if value >= 'A' && value <= 'Z' {
		return true
	}
	return value >= 'a' && value <= 'z'
}

func generateCode() (string, error) {
	buffer := make([]byte, codeBytes)
	value := new(big.Int)
	for attempt := 0; attempt < 8; attempt++ {
		if _, err := rand.Read(buffer); err != nil {
			return "", err
		}
		value.SetBytes(buffer)
		if value.Cmp(codeLimit) < 0 {
			return encodeBase62(value), nil
		}
	}
	return "", errors.New("failed to generate match invite code")
}

func encodeBase62(value *big.Int) string {
	digits := make([]byte, CodeLength)
	remaining := new(big.Int).Set(value)
	mod := new(big.Int)
	for index := CodeLength - 1; index >= 0; index-- {
		remaining.DivMod(remaining, codeBase, mod)
		digits[index] = codeAlphabet[mod.Int64()]
	}
	return string(digits)
}
