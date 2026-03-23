package utils

import (
	"context"
	"testing"
)

func TestGetOptionalUserIDFromCtxReturnsZeroForGuest(t *testing.T) {
	userID, err := GetOptionalUserIDFromCtx(context.Background())
	if err != nil {
		t.Fatalf("expected nil error for guest context, got %v", err)
	}
	if userID != 0 {
		t.Fatalf("expected guest user id 0, got %d", userID)
	}
}

func TestGetOptionalUserIDFromCtxReturnsUserIDWhenPresent(t *testing.T) {
	ctx := context.WithValue(context.Background(), "user_id", int64(123))

	userID, err := GetOptionalUserIDFromCtx(ctx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if userID != 123 {
		t.Fatalf("expected user id 123, got %d", userID)
	}
}
