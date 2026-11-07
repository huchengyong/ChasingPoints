package exportcursor

import (
	"errors"
	"testing"
)

func TestSignerBindsExportCursorClaimsAndRejectsTampering(t *testing.T) {
	signer, err := NewSigner("cursor-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	cursor, err := signer.Issue(Claims{
		FormatVersion:    "personal-data-export/v1",
		UserID:           42,
		SnapshotUnixNano: 1787212800000000000,
		Category:         "matches",
		LastID:           88,
	})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := signer.Verify(cursor)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != 42 || claims.SnapshotUnixNano != 1787212800000000000 || claims.Category != "matches" || claims.LastID != 88 {
		t.Fatalf("unexpected cursor claims: %+v", claims)
	}
	if _, err := signer.Verify(cursor[:len(cursor)-1] + "x"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("expected tampered cursor rejection, got %v", err)
	}
}

func TestSignerRejectsMissingSecretAndMalformedClaims(t *testing.T) {
	if _, err := NewSigner(""); !errors.Is(err, ErrSignerConfig) {
		t.Fatalf("expected missing signer secret rejection, got %v", err)
	}
	signer, err := NewSigner("cursor-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := signer.Issue(Claims{UserID: 1}); !errors.Is(err, ErrSignerConfig) {
		t.Fatalf("expected malformed claims rejection, got %v", err)
	}
}
