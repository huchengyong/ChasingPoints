package matchinvite

import (
	"context"
	"errors"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestCodeStore(t *testing.T) (*CodeStore, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	return NewCodeStore(redis.NewClient(&redis.Options{Addr: mr.Addr()})), mr
}

func TestCodeStoreSavesTokenWithRemainingTTL(t *testing.T) {
	store, mr := newTestCodeStore(t)

	code, err := store.Save(context.Background(), "signed.token", 5*time.Minute)
	if err != nil {
		t.Fatalf("save code: %v", err)
	}
	if !ValidCode(code) {
		t.Fatalf("generated code is not 12 base62 characters: %q", code)
	}

	key := codeKeyPrefix + code
	if !mr.Exists(key) {
		t.Fatalf("redis key missing: %s", key)
	}
	if mr.TTL(key) > 5*time.Minute || mr.TTL(key) <= 0 {
		t.Fatalf("unexpected ttl: %s", mr.TTL(key))
	}

	token, err := store.Resolve(context.Background(), code)
	if err != nil {
		t.Fatalf("resolve code: %v", err)
	}
	if token != "signed.token" {
		t.Fatalf("unexpected token: %s", token)
	}
}

func TestCodeStoreStopsWorkingAfterTTL(t *testing.T) {
	store, mr := newTestCodeStore(t)

	code, err := store.Save(context.Background(), "signed.token", time.Minute)
	if err != nil {
		t.Fatalf("save code: %v", err)
	}
	mr.FastForward(2 * time.Minute)

	if _, err := store.Resolve(context.Background(), code); !errors.Is(err, ErrCodeNotFound) {
		t.Fatalf("expected not found after expiry, got %v", err)
	}
}

func TestCodeStoreRejectsInvalidInput(t *testing.T) {
	store, _ := newTestCodeStore(t)
	ctx := context.Background()

	if _, err := store.Resolve(ctx, "short"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("expected invalid code, got %v", err)
	}
	if _, err := store.Resolve(ctx, "Ab3dEf6hIj9-"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("expected invalid code charset, got %v", err)
	}
	if _, err := store.Save(ctx, "", time.Minute); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("expected invalid token, got %v", err)
	}
	if _, err := store.Save(ctx, "signed.token", 0); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("expected invalid ttl, got %v", err)
	}
	if _, err := (*CodeStore)(nil).Save(ctx, "signed.token", time.Minute); !errors.Is(err, ErrCodeStoreMissing) {
		t.Fatalf("expected missing store, got %v", err)
	}
}

func TestGenerateCodesAreUniqueAndWellFormed(t *testing.T) {
	seen := make(map[string]struct{})
	for i := 0; i < 500; i++ {
		code, err := generateCode()
		if err != nil {
			t.Fatalf("generate code: %v", err)
		}
		if !ValidCode(code) {
			t.Fatalf("malformed code: %q", code)
		}
		if _, duplicate := seen[code]; duplicate {
			t.Fatalf("duplicate code generated: %s", code)
		}
		seen[code] = struct{}{}
	}
}
