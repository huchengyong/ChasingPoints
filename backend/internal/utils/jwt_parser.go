package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// GetUserIDFromCtx extracts the user ID from the go-zero context.
// It handles different numeric types that might be present after JWT decoding.
func GetUserIDFromCtx(ctx context.Context) (int64, error) {
	var userID int64
	var err error

	val := ctx.Value("user_id") // In this project, the key is "user_id"
	if val == nil {
		return 0, errors.New("unauthorized: user_id not found in context")
	}

	switch v := val.(type) {
	case json.Number:
		userID, err = v.Int64()
		if err != nil {
			return 0, fmt.Errorf("failed to parse user ID from json.Number: %w", err)
		}
	case float64:
		userID = int64(v)
	case int64:
		userID = v
	default:
		return 0, fmt.Errorf("invalid userId type '%T' in context", v)
	}

	if userID <= 0 {
		return 0, errors.New("invalid user ID in context")
	}

	return userID, nil
}

// GetOptionalUserIDFromCtx extracts user_id from context when present.
// It returns 0 with nil error for anonymous requests.
func GetOptionalUserIDFromCtx(ctx context.Context) (int64, error) {
	userID, err := GetUserIDFromCtx(ctx)
	if err != nil {
		return 0, nil
	}
	return userID, nil
}

// GetOptionalTokenTypeFromCtx extracts token_type from the JWT context when present.
// Legacy access tokens do not carry this claim and return an empty string.
func GetOptionalTokenTypeFromCtx(ctx context.Context) (string, error) {
	val := ctx.Value("token_type")
	if val == nil {
		return "", nil
	}
	tokenType, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("invalid token_type type '%T' in context", val)
	}
	return tokenType, nil
}

// GetOptionalSignedUserIDFromCtx extracts the raw signed user_id from context
// when present, preserving the sign. It returns 0 with nil error for anonymous
// requests; positive values are regular users and negative values are admin
// identities. Unlike GetUserIDFromCtx it does not reject zero or negative IDs,
// which makes it suitable for a global identity-aware middleware.
func GetOptionalSignedUserIDFromCtx(ctx context.Context) (int64, error) {
	val := ctx.Value("user_id")
	if val == nil {
		return 0, nil
	}

	switch v := val.(type) {
	case json.Number:
		userID, err := v.Int64()
		if err != nil {
			return 0, fmt.Errorf("failed to parse user ID from json.Number: %w", err)
		}
		return userID, nil
	case float64:
		return int64(v), nil
	case int64:
		return v, nil
	default:
		return 0, fmt.Errorf("invalid userId type '%T' in context", v)
	}
}

// GetAdminIDFromCtx extracts the admin ID from the context.
// Admin IDs are stored as negative values in JWT to distinguish from regular users.
func GetAdminIDFromCtx(ctx context.Context) (uint64, error) {
	var userID int64
	var err error

	val := ctx.Value("user_id")
	if val == nil {
		return 0, errors.New("unauthorized: user_id not found in context")
	}

	switch v := val.(type) {
	case json.Number:
		userID, err = v.Int64()
		if err != nil {
			return 0, fmt.Errorf("failed to parse admin ID from json.Number: %w", err)
		}
	case float64:
		userID = int64(v)
	case int64:
		userID = v
	default:
		return 0, fmt.Errorf("invalid userId type '%T' in context", v)
	}

	// Admin ID is stored as negative
	if userID >= 0 {
		return 0, errors.New("invalid admin ID in context")
	}

	return uint64(-userID), nil
}
