package utils

import (
	"context"
	"encoding/json"
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

func TestGetOptionalTokenTypeFromCtx(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		wantType string
		wantErr  bool
	}{
		{name: "missing token type", ctx: context.Background()},
		{name: "access token", ctx: context.WithValue(context.Background(), "token_type", "access"), wantType: "access"},
		{name: "refresh token", ctx: context.WithValue(context.Background(), "token_type", "refresh"), wantType: "refresh"},
		{name: "invalid token type claim", ctx: context.WithValue(context.Background(), "token_type", float64(1)), wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := GetOptionalTokenTypeFromCtx(tc.ctx)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if got != tc.wantType {
				t.Fatalf("expected token type %q, got %q", tc.wantType, got)
			}
		})
	}
}

func TestGetOptionalSignedUserIDFromCtx(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		wantID  int64
		wantErr bool
	}{
		{
			name:   "missing user id returns zero",
			ctx:    context.Background(),
			wantID: 0,
		},
		{
			name:   "positive int64 user id",
			ctx:    context.WithValue(context.Background(), "user_id", int64(123)),
			wantID: 123,
		},
		{
			name:   "negative int64 admin id keeps sign",
			ctx:    context.WithValue(context.Background(), "user_id", int64(-7)),
			wantID: -7,
		},
		{
			name:   "json number user id",
			ctx:    context.WithValue(context.Background(), "user_id", json.Number("456")),
			wantID: 456,
		},
		{
			name:   "json number negative admin id keeps sign",
			ctx:    context.WithValue(context.Background(), "user_id", json.Number("-9")),
			wantID: -9,
		},
		{
			name:   "float64 user id",
			ctx:    context.WithValue(context.Background(), "user_id", float64(88)),
			wantID: 88,
		},
		{
			name:    "invalid type errors",
			ctx:     context.WithValue(context.Background(), "user_id", "not-a-number"),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := GetOptionalSignedUserIDFromCtx(tc.ctx)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %d", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if got != tc.wantID {
				t.Fatalf("expected signed id %d, got %d", tc.wantID, got)
			}
		})
	}
}
