package matchinvite

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSignerIssuesAndVerifiesBoundShortLivedInvite(t *testing.T) {
	signer, err := NewSigner("test-secret", 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)

	token, claims, err := signer.IssueAt(42, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(strings.Split(token, ".")) != 2 || claims.Purpose != Purpose || claims.InviterUserID != 42 {
		t.Fatalf("unexpected invite: token=%q claims=%+v", token, claims)
	}
	if claims.Nonce == "" || claims.ExpiresAt != now.Add(5*time.Minute).Unix() {
		t.Fatalf("unexpected expiry claims: %+v", claims)
	}

	verified, err := signer.VerifyAt(token, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if *verified != *claims {
		t.Fatalf("verified claims mismatch: got=%+v want=%+v", verified, claims)
	}
}

func TestSignerRejectsTamperedAndExpiredInvite(t *testing.T) {
	signer, err := NewSigner("test-secret", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	token, _, err := signer.IssueAt(42, now)
	if err != nil {
		t.Fatal(err)
	}

	tampered := token[:len(token)-1] + "x"
	if _, err := signer.VerifyAt(tampered, now); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected tamper rejection, got %v", err)
	}
	if _, err := signer.VerifyAt(token, now.Add(time.Minute)); !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("expected expiry rejection, got %v", err)
	}
}

func TestSignerRequiresIndependentSecretAndCapsTTL(t *testing.T) {
	if _, err := NewSigner("", time.Minute); !errors.Is(err, ErrSignerConfig) {
		t.Fatalf("expected missing secret rejection, got %v", err)
	}
	signer, err := NewSigner("test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	_, claims, err := signer.IssueAt(42, now)
	if err != nil {
		t.Fatal(err)
	}
	if claims.ExpiresAt != now.Add(maxTTL).Unix() {
		t.Fatalf("ttl must be capped, got=%d", claims.ExpiresAt-now.Unix())
	}
}
