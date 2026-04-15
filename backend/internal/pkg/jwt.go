package pkg

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JwtClaims struct {
	UserId    int64  `json:"user_id"`
	TokenType string `json:"token_type,omitempty"`
	jwt.RegisteredClaims
}

// GenerateToken 生成JWT Token
func GenerateToken(userId int64, secret string, expireSeconds int64) (string, error) {
	return GenerateTypedToken(userId, secret, expireSeconds, "")
}

func GenerateTypedToken(userId int64, secret string, expireSeconds int64, tokenType string) (string, error) {
	claims := JwtClaims{
		UserId:    userId,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireSeconds) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
