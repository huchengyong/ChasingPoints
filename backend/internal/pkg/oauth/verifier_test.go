package oauth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestStaticVerifierReturnsNormalizedIdentity(t *testing.T) {
	verifier := StaticVerifier{Identity: &Identity{Provider: " HUAWEI ", Subject: " subject-1 ", UnionID: " union-1 "}}

	identity, err := verifier.Verify(context.Background(), VerifyRequest{Provider: "huawei"})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if identity.Provider != ProviderHuawei || identity.Subject != "subject-1" || identity.UnionID != "union-1" {
		t.Fatalf("unexpected identity: %+v", identity)
	}
}

func TestStaticVerifierReturnsStableRejectedError(t *testing.T) {
	errRejected := NewVerifyError(ErrorRejected, "rejected", nil)
	verifier := StaticVerifier{Err: errRejected}

	_, err := verifier.Verify(context.Background(), VerifyRequest{Provider: "huawei"})
	if !errors.Is(err, errRejected) || CategoryOf(err) != ErrorRejected {
		t.Fatalf("expected rejected verifier error, got %v", err)
	}
}

func TestStaticVerifierCanReturnTimeoutCategory(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	<-ctx.Done()

	verifier := StaticVerifier{Delay: func(context.Context) error {
		return NewVerifyError(ErrorUnavailable, "timeout", ctx.Err())
	}}

	_, err := verifier.Verify(ctx, VerifyRequest{Provider: "huawei"})
	if CategoryOf(err) != ErrorUnavailable {
		t.Fatalf("expected unavailable timeout category, got %v", err)
	}
}

func TestStaticVerifierRejectsInvalidIdentity(t *testing.T) {
	verifier := StaticVerifier{Identity: &Identity{Provider: "huawei"}}

	_, err := verifier.Verify(context.Background(), VerifyRequest{Provider: "huawei"})
	if CategoryOf(err) != ErrorInvalidResponse {
		t.Fatalf("expected invalid response category, got %v", err)
	}
}
