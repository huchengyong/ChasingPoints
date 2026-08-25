package exportcursor

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

var (
	ErrInvalidCursor = errors.New("invalid personal data export cursor")
	ErrSignerConfig  = errors.New("invalid personal data export cursor signer config")
)

type Claims struct {
	FormatVersion    string `json:"format_version"`
	UserID           int64  `json:"user_id"`
	SnapshotUnixNano int64  `json:"snapshot_unix_nano"`
	Category         string `json:"category"`
	LastID           int64  `json:"last_id"`
}

type Signer struct {
	secret []byte
}

func NewSigner(secret string) (*Signer, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, ErrSignerConfig
	}
	derived := sha256.Sum256([]byte("personal-data-export-cursor:" + secret))
	return &Signer{secret: derived[:]}, nil
}

func (s *Signer) Issue(claims Claims) (string, error) {
	if s == nil || len(s.secret) == 0 || claims.UserID <= 0 || claims.SnapshotUnixNano <= 0 ||
		strings.TrimSpace(claims.FormatVersion) == "" || strings.TrimSpace(claims.Category) == "" || claims.LastID < 0 {
		return "", ErrSignerConfig
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	return encodedPayload + "." + base64.RawURLEncoding.EncodeToString(s.sign(encodedPayload)), nil
}

func (s *Signer) Verify(cursor string) (*Claims, error) {
	if s == nil || len(s.secret) == 0 {
		return nil, ErrSignerConfig
	}
	parts := strings.Split(strings.TrimSpace(cursor), ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, ErrInvalidCursor
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || subtle.ConstantTimeCompare(signature, s.sign(parts[0])) != 1 {
		return nil, ErrInvalidCursor
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrInvalidCursor
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil || claims.UserID <= 0 || claims.SnapshotUnixNano <= 0 ||
		strings.TrimSpace(claims.FormatVersion) == "" || strings.TrimSpace(claims.Category) == "" || claims.LastID < 0 {
		return nil, ErrInvalidCursor
	}
	return &claims, nil
}

func (s *Signer) sign(payload string) []byte {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(payload))
	return mac.Sum(nil)
}
