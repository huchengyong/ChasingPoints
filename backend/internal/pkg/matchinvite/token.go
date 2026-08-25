package matchinvite

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const (
	Purpose       = "match_invite"
	defaultTTL    = 5 * time.Minute
	maxTTL        = 15 * time.Minute
	nonceByteSize = 16
)

var (
	ErrInvalidToken = errors.New("invalid match invite token")
	ErrExpiredToken = errors.New("expired match invite token")
	ErrSignerConfig = errors.New("invalid match invite signer config")
)

type Claims struct {
	Purpose       string `json:"purpose"`
	InviterUserID int64  `json:"inviter_user_id"`
	IssuedAt      int64  `json:"issued_at"`
	ExpiresAt     int64  `json:"expires_at"`
	Nonce         string `json:"nonce"`
}

type Signer struct {
	secret []byte
	ttl    time.Duration
}

func NewSigner(secret string, ttl time.Duration) (*Signer, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, ErrSignerConfig
	}
	if ttl <= 0 {
		ttl = defaultTTL
	}
	if ttl > maxTTL {
		ttl = maxTTL
	}
	return &Signer{secret: []byte(secret), ttl: ttl}, nil
}

func (s *Signer) Issue(inviterUserID int64) (string, *Claims, error) {
	return s.IssueAt(inviterUserID, time.Now())
}

func (s *Signer) IssueAt(inviterUserID int64, now time.Time) (string, *Claims, error) {
	if s == nil || len(s.secret) == 0 || inviterUserID <= 0 {
		return "", nil, ErrSignerConfig
	}

	nonce, err := randomNonce()
	if err != nil {
		return "", nil, err
	}
	claims := &Claims{
		Purpose:       Purpose,
		InviterUserID: inviterUserID,
		IssuedAt:      now.Unix(),
		ExpiresAt:     now.Add(s.ttl).Unix(),
		Nonce:         nonce,
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", nil, err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := s.sign(encodedPayload)
	return encodedPayload + "." + base64.RawURLEncoding.EncodeToString(signature), claims, nil
}

func (s *Signer) Verify(token string) (*Claims, error) {
	return s.VerifyAt(token, time.Now())
}

func (s *Signer) VerifyAt(token string, now time.Time) (*Claims, error) {
	if s == nil || len(s.secret) == 0 {
		return nil, ErrSignerConfig
	}
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, ErrInvalidToken
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || subtle.ConstantTimeCompare(signature, s.sign(parts[0])) != 1 {
		return nil, ErrInvalidToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrInvalidToken
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, ErrInvalidToken
	}
	if claims.Purpose != Purpose || claims.InviterUserID <= 0 || claims.Nonce == "" ||
		claims.IssuedAt <= 0 || claims.ExpiresAt <= claims.IssuedAt {
		return nil, ErrInvalidToken
	}
	if now.Unix() >= claims.ExpiresAt {
		return nil, ErrExpiredToken
	}
	return &claims, nil
}

func (s *Signer) sign(payload string) []byte {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(payload))
	return mac.Sum(nil)
}

func randomNonce() (string, error) {
	value := make([]byte, nonceByteSize)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
